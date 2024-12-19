package v1

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

const PROJECT_DIRS = "projects"

func RouteCompose(group *echo.Group) {
	group.POST("/:name", createProject)
	group.GET("/:name", getProject)
	group.GET("/:name/status", getStatus)
	group.POST("/:name/start", startProject)
	group.POST("/:name/stop", stopProject)
	group.DELETE("/:name", deleteProject)
}

type ComposeProjectYaml struct {
	Version  string                 `yaml:"version,omitempty" json:"version,omitempty"`
	Services map[string]interface{} `yaml:"services" json:"services"`
	Volumes  map[string]interface{} `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks map[string]interface{} `yaml:"networks,omitempty" json:"networks,omitempty"`
}

func createProject(c echo.Context) error {
	// read yaml file
	log.Printf("[DEBUG] create project %s", c.Param("name"))
	name := c.Param("name")
	body := ComposeProjectYaml{}
	err := c.Bind(&body)
	if err != nil {
		return c.JSON(400, map[string]string{
			"message": err.Error(),
		})
	}
	if body.Services == nil {
		body.Services = map[string]interface{}{}
	}

	projectPath := path.Join(PROJECT_DIRS, name)

	// create dir
	err = os.MkdirAll(projectPath, 0755)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	// create docker-compose.yml
	composeFilePath := filepath.Join(projectPath, "docker-compose.yml")
	file, err := os.OpenFile(composeFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return c.JSON(500, map[string]string{
			"when":    "create docker-compose.yml",
			"message": err.Error(),
		})
	}
	defer file.Close()
	// write yaml file, replace existing
	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	err = enc.Encode(body)
	if err != nil {
		return c.JSON(500, map[string]string{
			"when":    "write docker-compose.yml",
			"message": err.Error(),
		})
	}

	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func getProject(c echo.Context) error {
	name := c.Param("name")
	projectDir := filepath.Join(PROJECT_DIRS, name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	// go to project directory and trigger docker compose up
	cmd := exec.Command("docker-compose", "config")
	cmd.Dir = projectDir
	// read from cmd output
	buff := bytes.NewBuffer([]byte{})
	cmd.Stdout = buff
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	prj := map[string]interface{}{}
	err = yaml.NewDecoder(buff).Decode(&prj)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	log.Printf("[DEBUG] prj %+v", prj)
	return c.JSON(200, prj)
}

type ExpectedPSData struct {
	Service    string `json:"Service"`
	CreatedAt  string `json:"CreatedAt"`
	Image      string `json:"Image"`
	Status     string `json:"Status"`
	State      string `json:"State"`
	Size       string `json:"Size"`
	RunningFor string `json:"RunningFor"`
	ExitCode   int    `json:"ExitCode"`
}

func getStatus(c echo.Context) error {
	name := c.Param("name")
	all := c.QueryParam("all")
	projectDir := filepath.Join(PROJECT_DIRS, name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	// go to project directory and trigger docker compose ps
	commands := []string{"ps", "--format", "json"}
	if all == "true" {
		commands = append(commands, "-a")
	}

	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			commands = append(commands, v...)
			continue
		}
	}

	cmd := exec.Command("docker-compose", commands...)
	cmd.Dir = projectDir
	// read from cmd output
	buff := bytes.NewBuffer([]byte{})
	cmd.Stdout = buff
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	outputStrs := strings.Split(buff.String(), "\n")
	services := map[string]ExpectedPSData{}
	for _, outputStr := range outputStrs {
		if outputStr == "" {
			continue
		}
		psData := ExpectedPSData{}
		if err := json.Unmarshal([]byte(outputStr), &psData); err != nil {
			return c.JSON(500, map[string]string{
				"message": err.Error(),
				"txt":     outputStr,
			})
		}
		services[psData.Service] = psData
	}
	return c.JSON(200, map[string]interface{}{
		"services": services,
	})
}

func startProject(c echo.Context) error {
	name := c.Param("name")
	restart := c.QueryParam("restart")
	pull := c.QueryParam("pull")
	projectDir := filepath.Join(PROJECT_DIRS, name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}

	// go to project directory and trigger docker compose up
	var cmd *exec.Cmd
	commands := []string{"up", "-d"}
	if restart == "true" {
		commands = append(commands, "--force-recreate")
	}
	if pull == "true" {
		commands = append(commands, "--pull", "always")
	}

	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			commands = append(commands, v...)
			continue
		}
	}
	cmd = exec.Command("docker-compose", commands...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func stopProject(c echo.Context) error {
	name := c.Param("name")
	projectDir := filepath.Join(name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}

	commands := []string{"stop"}
	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			commands = append(commands, v...)
			continue
		}
	}

	// go to project directory and trigger docker compose up
	cmd := exec.Command("docker-compose", commands...)

	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func deleteProject(c echo.Context) error {
	name := c.Param("name")
	projectDir := filepath.Join(PROJECT_DIRS, name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}

	// go to project directory and trigger docker compose up
	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}
