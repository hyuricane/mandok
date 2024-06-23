package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
	mTypes "inovasiriset.co.id/docker/manager/types"
)

var dockerCli *client.Client // docker client
var registryAuths map[string]registry.AuthConfig
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
	registryAuths = map[string]registry.AuthConfig{}
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
			registryAuths[segments[1]] = registry.AuthConfig{
				Username: segments[0][:i],
				Password: segments[0][i+1:],
			}
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
		return c.JSON(200, containers[0])
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
		regauth := ""
		for k, v := range registryAuths {
			if strings.HasPrefix(containers[0].Image, k) {
				log.Printf("[DEBUG] use registry auth %s", k)
				encodedJSON, err := json.Marshal(v)
				if err != nil {
					return err
				}
				regauth = base64.URLEncoding.EncodeToString(encodedJSON)
				break
			}
		}
		pullResp, err := dockerCli.ImagePull(context.TODO(), containers[0].Image, image.PullOptions{RegistryAuth: regauth})
		if err != nil {
			log.Printf("[DEBUG] pull image %s error:  %v", containers[0].Image, err)
		} else {
			defer pullResp.Close()
			pullRespStr, err := io.ReadAll(pullResp)
			log.Printf("[DEBUG] pull image %s response: %s -- %v", containers[0].Image, pullRespStr, err)
		}
		// update container with newly pulled image
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
		// docker compose command
		dockerComposeCmdName := os.Getenv("DOCKER_COMPOSE_CMD_NAME")
		if dockerComposeCmdName == "" {
			dockerComposeCmdName = "docker-compose"
		}
		dockerComposeArgs := []string{"up", "-d", "--force-recreate", containerName}
		dockerComposeCmdNames := strings.Split(dockerComposeCmdName, " ")
		if len(dockerComposeCmdNames) > 0 {
			dockerComposeCmdName = dockerComposeCmdNames[0]
			dockerComposeArgs = append(dockerComposeCmdNames[1:], dockerComposeArgs...)
		}
		log.Printf("[DEBUG] cmd: %s %v", dockerComposeCmdName, dockerComposeArgs)
		cmd := exec.Command(dockerComposeCmdName, dockerComposeArgs...)
		cmd.Dir = workdir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Printf("[ERROR] restart container error: %v", err)
			return err
		}
		// log.Printf("[DEBUG] container %s updated: %v", containers[0].ID, updateOk)
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
