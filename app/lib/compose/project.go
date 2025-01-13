package compose

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ComposeProjectYaml struct {
	Name     string                            `yaml:"-" json:"name,omitempty"`
	Version  string                            `yaml:"version,omitempty" json:"version,omitempty"`
	Services map[string]map[string]interface{} `yaml:"services" json:"services"`
	Volumes  map[string]interface{}            `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks map[string]interface{}            `yaml:"networks,omitempty" json:"networks,omitempty"`
}

func CreateProject(name string, projectConfig ComposeProjectYaml) (string, error) {
	if name == "" {
		return "", nil
	}

	if projectConfig.Services == nil {
		projectConfig.Services = map[string]map[string]interface{}{}
	}
	// create dir
	projectPath := path.Join(PROJECT_DIRS, name)
	err := os.MkdirAll(projectPath, 0755)
	if err != nil {
		return "", err
	}
	// create docker-compose.yml
	composeFilePath := filepath.Join(projectPath, "docker-compose-tmp.yml")
	file, err := os.OpenFile(composeFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// write yaml file, replace existing
	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	err = enc.Encode(projectConfig)
	if err != nil {
		return "", err
	}
	if len(projectConfig.Services) == 0 {
		err = os.Rename(composeFilePath, filepath.Join(projectPath, "docker-compose.yml"))
		if err != nil {
			return "", err
		}
		return projectPath, nil
	}
	err = TryProject(projectPath, "docker-compose-tmp.yml")
	if err != nil {
		return "", err
	}
	// move docker-compose-tmp.yml to docker-compose.yml
	err = os.Rename(composeFilePath, filepath.Join(projectPath, "docker-compose.yml"))
	if err != nil {
		return "", err
	}
	return projectPath, nil
}

func HasProject(name string) string {
	projectPath := path.Join(PROJECT_DIRS, name)
	projectDir := filepath.Join(PROJECT_DIRS, name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return ""
	} else if err != nil {
		log.Printf("[ERROR] hasProject %v", err)
		return ""
	}
	return projectPath
}

func GetProject(projectDir string, nointerpolate ...bool) (*ComposeProjectYaml, error) {
	// go to project directory and trigger docker compose up
	args := []string{"--env-file", "masked.env", "config", "--format", "json"}
	if len(nointerpolate) > 0 && nointerpolate[0] {
		args = []string{"config", "--format", "json", "--no-interpolate"}
	}
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir
	out, err := doExec(cmd)
	if err != nil {
		return nil, NewComposeError(bytes.NewBufferString(err.Error()))
	}
	if out == nil {
		return nil, nil
	}
	prj := ComposeProjectYaml{}
	err = json.NewDecoder(out).Decode(&prj)
	if err != nil {
		return nil, err
	}
	return &prj, nil
}

func GetProjects() ([]string, error) {
	projects := []string{}
	files, err := os.ReadDir(PROJECT_DIRS)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			projects = append(projects, file.Name())
		}
	}
	return projects, nil
}
