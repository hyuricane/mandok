package v1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
	mTypes "inovasiriset.co.id/docker/manager/types"
)

var dockerCli *client.Client // docker client
var registryAuths map[string]string
var workdirs map[string]string

func init() {
	godotenv.Load()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("[ERROR] new docker client error: %v", err)
		log.Fatal(err)
	}
	dockerCli = cli
	log.Printf("[DEBUG] docker client initialized")
	registryAuths = map[string]string{}
	if regauthstr := os.Getenv("REGISTRY_AUTHS"); regauthstr != "" {
		regauths := strings.Split(regauthstr, ",")
		for _, auth := range regauths {
			auth = strings.TrimSpace(auth)
			if auth == "" {
				continue
			}
			segments := strings.Split(auth, "@")
			if len(segments) != 2 {
				continue
			}
			i := strings.Index(segments[0], ":")
			if i == -1 {
				continue
			}
			encodedJSON, err := json.Marshal(map[string]string{
				"username": segments[0][:i],
				"password": segments[0][i+1:],
			})
			if err != nil {
				continue
			}
			regauth := base64.URLEncoding.EncodeToString(encodedJSON)
			registryAuths[segments[1]] = regauth
		}
	}
	workdirs = map[string]string{}
	if workdirsStr := os.Getenv("WORKDIRS"); workdirsStr != "" {
		segments := strings.Split(workdirsStr, ";")

		for _, segment := range segments {
			if segment == "" {
				continue
			}
			workdirss := strings.Split(segment, ":")
			if len(workdirss) != 2 {
				continue
			}
			workdirs[strings.TrimSpace(workdirss[0])] = strings.TrimSpace(workdirss[1])
		}
	}
}

func RouteDocker(group *echo.Group) {
	group.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, Docker!")
	})
	group.GET("/containers", GetContainers)
	group.GET("/containers/:project", GetContainers)
	group.GET("/containers/:project/:container", DetailContainer)
	group.PUT("/containers/:project/:container", UpdateContainer)
	group.GET("/containers/:project/:container/restart", RestartContainer)
	// group.POST("/containers/:project/:container/restart", RestartContainerPost)
}

func GetContainers(c echo.Context) error {
	projectName := c.Param("project")
	containers, err := getContainers(projectName, "")
	if err != nil {
		log.Printf("[ERROR] get containers error: %v", err)
		return err
	}
	resp := map[string]interface{}{}
	projects := map[string][]map[string]interface{}{}
	standalone := []map[string]interface{}{}

	for i := 0; i < len(containers); i++ {
		if prjname, ok := containers[i].Labels["com.docker.compose.project"]; ok {
			if _, ok := projects[prjname]; !ok {
				projects[prjname] = []map[string]interface{}{
					{
						"name":   containers[i].Labels["com.docker.compose.service"],
						"image":  containers[i].Image,
						"status": containers[i].Status,
					},
				}
			} else {
				projects[prjname] = append(projects[prjname], map[string]interface{}{
					"name":   containers[i].Labels["com.docker.compose.service"],
					"image":  containers[i].Image,
					"status": containers[i].Status,
				})
			}
		} else {
			standalone = append(standalone, map[string]interface{}{
				"name":   containers[i].Names[0],
				"image":  containers[i].Image,
				"status": containers[i].Status,
			})
		}
	}
	resp["projects"] = projects
	resp["standalone"] = standalone
	return c.JSON(200, resp)
}

func DetailContainer(c echo.Context) error {
	projectName := c.Param("project")
	containerName := c.Param("container")

	// find container
	containers, err := getContainers(projectName, containerName)
	if err != nil {
		log.Printf("[ERROR] get containers error: %v", err)
		return err
	}
	if len(containers) == 0 {
		return c.JSON(200, map[string]string{
			"message": "container not found",
		})
	}
	if len(containers) == 1 {
		rpid, err := dockerCli.ContainerExecCreate(context.TODO(), containers[0].ID, container.ExecOptions{
			Cmd: []string{"printenv"},
		})
		if err != nil {
			log.Printf("[ERROR] exec error: %v", err)
			return err
		}
		rp, err := dockerCli.ContainerExecAttach(context.TODO(), rpid.ID, container.ExecStartOptions{})
		if err != nil {
			log.Printf("[ERROR] attach error: %v", err)
			return err
		}
		defer rp.Close()
		_, err = io.ReadAll(rp.Reader)
		if err != nil {
			log.Printf("[ERROR] read error: %v", err)
			return err
		}
		return c.JSON(200, map[string]interface{}{
			"ID":      containers[0].ID,
			"Name":    containers[0].Names[0],
			"Image":   containers[0].Image,
			"ImageID": containers[0].ImageID,
			"Status":  containers[0].Status,
			"State":   containers[0].State,
		})
	}
	return c.JSON(200, map[string]interface{}{
		"message":    "too many containers found",
		"containers": containers,
	})
}

func UpdateContainer(c echo.Context) error {
	projectName := c.Param("project")
	containerName := c.Param("container")

	// find container
	containers, err := getContainers(projectName, containerName)
	if err != nil {
		log.Printf("[ERROR] get containers error: %v", err)
		return err
	}
	if len(containers) == 0 {
		return c.JSON(200, map[string]string{
			"message": "container not found",
		})
	}
	if len(containers) > 1 {
		return c.JSON(200, map[string]string{
			"message": "too many containers found",
		})
	}

	// read docker compose file in com.docker.compose.project.config_file
	configFilePath, ok := containers[0].Labels["com.docker.compose.project.config_files"]
	if !ok {
		return c.JSON(200, map[string]string{
			"message": "no config file found",
		})
	}
	configFilePath = strings.TrimSpace(configFilePath)
	if configFilePath == "" {
		return c.JSON(200, map[string]string{
			"message": "empty config file",
		})
	}

	configFileBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Printf("[ERROR] read config file error: %v", err)
		return err
	}
	projectConfig := mTypes.ProjectConfig{}
	err = yaml.Unmarshal(configFileBytes, &projectConfig)
	if err != nil {
		log.Printf("[ERROR] unmarshal config file error: %v", err)
		return err
	}

	// read service config from body
	inputServiceConfig := mTypes.ServiceConfig{}
	err = c.Bind(&inputServiceConfig)
	if err != nil {
		log.Printf("[ERROR] bind service config error: %v", err)
		return err
	}

	serviceConfig, ok := projectConfig.Services[containers[0].Labels["com.docker.compose.service"]]
	if !ok {
		return c.JSON(200, map[string]string{
			"message": "no service found",
		})
	}

	// merge service config

	log.Printf("[DEBUG] service config: %+v", serviceConfig)

	// write to file
	dir := filepath.Dir(configFilePath)
	tempConfigFilePath := filepath.Join(dir, "mandok-"+filepath.Base(configFilePath))

	tempProjectConfig := mTypes.ProjectConfig{
		Services: make(map[string]mTypes.ServiceConfig),
	}

	// read if already exists
	if _, err := os.Stat(tempConfigFilePath); err == nil {
		// file exists
		configFileBytes, err := os.ReadFile(tempConfigFilePath)
		if err != nil {
			log.Printf("[ERROR] read temp config file error: %v", err)
			return err
		}
		err = yaml.Unmarshal(configFileBytes, &tempProjectConfig)
		if err != nil {
			log.Printf("[ERROR] unmarshal temp config file error: %v", err)
			return err
		}
	}
	tempProjectConfig.Services[containerName] = serviceConfig

	log.Printf("[DEBUG] tempProjectConfig %+v", tempProjectConfig)

	tempProjectConfigBytes, err := yaml.Marshal(tempProjectConfig)
	if err != nil {
		log.Printf("[ERROR] marshal service config error: %v", err)
		return err
	}
	err = os.WriteFile(tempConfigFilePath, tempProjectConfigBytes, 0644)
	if err != nil {
		log.Printf("[ERROR] write service config error: %v", err)
		return err
	}

	// restart container
	return c.JSON(200, serviceConfig)
}

func RestartContainer(c echo.Context) error {
	projectName := c.Param("project")
	containerName := c.Param("container")

	// find container
	containers, err := getContainers(projectName, containerName)
	if err != nil {
		log.Printf("[ERROR] get containers error: %v", err)
		return err
	}
	if len(containers) == 0 {
		return c.JSON(200, map[string]string{
			"message": "container not found",
		})
	}

	if len(containers) == 1 {
		workdir, ok := workdirs[projectName]
		if !ok {
			workdir, ok = containers[0].Labels["com.docker.compose.project.working_dir"]
		}
		if !ok {
			return c.JSON(200, map[string]string{
				"project": projectName,
				"name":    containerName,
				"message": "not restarted",
			})
		}
		if c.Request().Header.Get("X-Image-Tag") != "" {
			err = updateImageTagEnv(filepath.Join(workdir, ".env"), containerName, c.Request().Header.Get("X-Image-Tag"))
			if err != nil {
				log.Printf("[ERROR] update image tag error: %v", err)
				return err
			}
		}
		// docker compose command
		err = restartContainer(workdir, containerName, strings.Split(containers[0].Labels["com.docker.compose.project.config_files"], ",")...)
		if err != nil {
			log.Printf("[ERROR] restart container error: %v", err)
			return err
		}
		// log.Printf("[DEBUG] container %s updated", containers[0].ID)
		return c.JSON(200, map[string]string{

			"project": projectName,
			"name":    containerName,
			"image":   containers[0].Image,
			"status":  "restarted",
		})
	}
	return c.JSON(200, map[string]string{
		"message": "too many containers found",
	})
}

func getContainers(projectName string, containerName string) ([]types.Container, error) {
	listFilters := filters.NewArgs()
	// listFilters.Add("label", "com.docker.compose.project=")
	listFilters.Add("label", "mandok=visible")
	if projectName == "standalone" {
		if containerName != "" {
			listFilters.Add("name", containerName)
		}
	} else if projectName != "" {
		listFilters.Add("label", "com.docker.compose.project="+projectName)
		if containerName != "" {
			listFilters.Add("label", "com.docker.compose.service="+containerName)
		}
	}
	return dockerCli.ContainerList(context.TODO(), container.ListOptions{Filters: listFilters})
}

func restartContainer(workdir string, containerName string, configFiles ...string) error {
	// Build docker compose command arguments
	cmdName := os.Getenv("DOCKER_COMPOSE_CMD_NAME")
	if cmdName == "" {
		cmdName = "docker compose"
	}
	args := []string{}
	cleanEnv := cleanEnvVars()

	projectName := ""
	// Add config files
	for _, configFile := range configFiles {
		args = append(args, "-f", configFile)
		configFileBytes, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("[ERROR] read config file error: %v", err)
			return err
		}
		configFileYaml := yaml.NewDecoder(bytes.NewReader(configFileBytes))
		var config map[string]any
		configFileYaml.Decode(&config)
		if name, ok := config["name"].(string); ok {
			projectName = name
		}
	}
	if projectName == "" {
		projectName = filepath.Base(workdir)
	}
	args = append(args, "-p", projectName)

	// Pull the latest image first
	pullArgs := append(args, "pull", containerName)
	// split cmdName
	cmdNameSplit := strings.Split(cmdName, " ")
	pullCmd := exec.Command(cmdNameSplit[0], append(cmdNameSplit[1:], pullArgs...)...)
	pullCmd.Dir = workdir
	pullCmd.Env = cleanEnv
	pullOut, pullErr := pullCmd.CombinedOutput()
	if pullErr != nil {
		log.Printf("[DEBUG] pull command: %s", pullCmd.String())
		log.Printf("[DEBUG] pull output: %s", string(pullOut))
		log.Printf("[WARNING] pull error (continuing anyway): %v", pullErr)
		// Continue even if pull fails - the image might already exist
	}

	// Restart the container using up with force-recreate
	upArgs := append(args, "up", "-d", "--force-recreate", containerName)
	upCmd := exec.Command(cmdNameSplit[0], append(cmdNameSplit[1:], upArgs...)...)
	upCmd.Dir = workdir
	upCmd.Env = cleanEnv
	upOut, upErr := upCmd.CombinedOutput()
	if upErr != nil {
		log.Printf("[DEBUG] up command: %s", upCmd.String())
		log.Printf("[DEBUG] up output: %s", string(upOut))
		log.Printf("[WARNING] up error (continuing anyway): %v", upErr)
		return upErr
	}

	log.Printf("[DEBUG] container %s restarted successfully", containerName)
	return nil
}

func updateImageTagEnv(envPath string, containerName string, imageTag string) error {
	// Append timestamped image tag update to .env file
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	envLine := fmt.Sprintf("# Updated %s\nIMAGE_TAG_%s=%s\n",
		timestamp,
		containerName,
		imageTag)
	var err error

	// Append to .env file (creates if doesn't exist)
	if _, err = os.Stat(envPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[ERROR] failed to write .env file: %v", err)
		return err
	}

	// If file didn't exist, this creates it. If it did, we need to append
	if os.IsNotExist(err) {
		if err := os.WriteFile(envPath, []byte(envLine), 0o644); err != nil {
			log.Printf("[ERROR] failed to create .env file: %v", err)
			return err
		}
	} else {
		// File exists, append to it
		f, err := os.OpenFile(envPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("[ERROR] failed to open .env for append: %v", err)
			return err
		}
		defer f.Close()

		if _, err := f.WriteString(envLine + "\n"); err != nil {
			log.Printf("[ERROR] failed to append to .env: %v", err)
			return err
		}
	}

	log.Printf("[INFO] Appended to .env: IMAGE_TAG_%s=%s at %s",
		containerName,
		imageTag,
		timestamp)
	return nil
}

func cleanEnvVars() []string {
	env := os.Environ()
	ommitedKeys := []string{
		"DOCKER_HOST",
		"REGISTRY_AUTHS",
		"API_USERNAME",
		"API_PASSWORD",
		"PORT",
		"IP",
		"BASE_URL",
		"DOCKER_COMPOSE_CMD_NAME",
		"WORKDIRS",
		"REGISTRY_HOST",
		"REGISTRY_USERNAME",
		"REGISTRY_PASSWORD",
		"MANDOK_DOMAIN",
	}
	result := []string{}
nextVar:
	for _, evar := range env {
		for _, key := range ommitedKeys {
			if strings.HasPrefix(evar, key+"=") {
				continue nextVar
			}
		}
		result = append(result, evar)
	}
	return result
}
