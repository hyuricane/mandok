package v1

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

const PROJECT_DIRS = "projects"

func RouteComposeV2(group *echo.Group) {
	group.POST("/:name", createProject)
	group.GET("/:name", getProject)
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

// type ComposeServiceYaml struct {
// 	Image    string            `yaml:"image" json:"image"`
// 	Ports    []ComposePortYaml `yaml:"ports,omitempty" json:"ports,omitempty"`
// 	Env      []string          `yaml:"environment,omitempty" json:"environment,omitempty"`
// 	Labels   map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
// 	Volumes  []string          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
// 	Depends  []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
// 	Networks []string          `yaml:"networks,omitempty" json:"networks,omitempty"`
// 	Command  string            `yaml:"command,omitempty" json:"command,omitempty"`
// 	Restart  string            `yaml:"restart,omitempty" json:"restart,omitempty"`
// }

// type ComposeVolumeYaml struct {
// 	Driver string `yaml:"driver,omitempty" json:"driver,omitempty"`
// }

// type ComposePortYaml struct {
// 	HostIP    string `yaml:"host_ip,omitempty" json:"host_ip,omitempty"`
// 	Mode      string `yaml:"mode,omitempty" json:"mode,omitempty"`
// 	Published string `yaml:"published,omitempty" json:"published,omitempty"`
// 	Target    string `yaml:"target,omitempty" json:"target,omitempty"`
// 	Protocol  string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
// }

// func (c *ComposePortYaml) unmarshalMap(m map[string]interface{}) {
// 	if vI, ok := m["mode"]; ok {
// 		if v, ok := vI.(string); ok {
// 			c.Mode = v
// 		}
// 	}
// 	if vI, ok := m["host_ip"]; ok {
// 		if v, ok := vI.(string); ok {
// 			c.HostIP = v
// 		}
// 	}
// 	if vI, ok := m["published"]; ok {
// 		if v, ok := vI.(string); ok {
// 			c.Published = v
// 		}
// 	}
// 	if vI, ok := m["target"]; ok {
// 		if v, ok := vI.(string); ok {
// 			c.Target = v
// 		}
// 	}
// 	if vI, ok := m["protocol"]; ok {
// 		if v, ok := vI.(string); ok {
// 			c.Protocol = v
// 		}
// 	}
// 	if c.Mode == "" {
// 		c.Mode = "ingress"
// 	}
// 	if c.Protocol == "" {
// 		c.Protocol = "tcp"
// 	}
// }

// func (c *ComposePortYaml) unmarshalString(s string) {
// 	p0 := strings.Split(s, "/")
// 	if len(p0) == 2 {
// 		c.Protocol = p0[1]
// 	} else {
// 		c.Protocol = "tcp"
// 	}
// 	pI := strings.LastIndex(p0[0], ":")
// 	p1 := strings.Split(p0[0], ":")
// 	if len(p1) == 3 {
// 		c.HostIP = p1[0]
// 		c.Published = p1[1]
// 		c.Target = p1[2]
// 	} else if len(p1) == 2 {
// 		c.Published = p1[0]
// 		c.Target = p1[1]
// 	} else if pI == -1 {
// 		c.Target = p0[0]
// 	}
// 	c.Mode = "ingress"
// }

// func (c *ComposePortYaml) UnmarshalYAML(value *yaml.Node) error {
// 	if value == nil {
// 		return nil
// 	}
// 	if value.Kind == yaml.MappingNode {
// 		m := map[string]interface{}{}
// 		if err := value.Decode(&m); err != nil {
// 			return err
// 		}
// 		c.unmarshalMap(m)
// 	} else if value.Kind == yaml.ScalarNode { // is string
// 		c.unmarshalString(value.Value)
// 	}
// 	return nil
// }

// func (c *ComposePortYaml) UnmarshalJSON(payload []byte) error {
// 	if len(payload) == 0 {
// 		return nil
// 	}
// 	if payload[0] == '{' {
// 		m := map[string]interface{}{}
// 		if err := yaml.Unmarshal(payload, &m); err != nil {
// 			return err
// 		}
// 		c.unmarshalMap(m)
// 	} else if payload[0] == '"' {
// 		vstr := string(payload[1 : len(payload)-1])
// 		c.unmarshalString(vstr)
// 	}
// 	return nil
// }

// type ComposeNetworkYaml struct {
// 	Driver string `yaml:"driver,omitempty" json:"driver,omitempty"`
// }

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
	log.Printf("[DEBUG] %s", buff.String())
	err = yaml.NewDecoder(buff).Decode(&prj)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	log.Printf("[DEBUG] prj %+v", prj)
	return c.JSON(200, prj)
}

func startProject(c echo.Context) error {
	name := c.Param("name")
	restart := c.QueryParam("restart")
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
