package compose

import (
	"encoding/json"
	"os/exec"
	"path"
	"strings"
)

func GetService(projectDir string, service string, nointerpolate ...bool) (map[string]interface{}, error) {
	// go to project directory and trigger docker compose up
	args := []string{"--env-file", "masked.env", "config", "--format", "json", service}
	if len(nointerpolate) > 0 && nointerpolate[0] {
		args = []string{"config", "--format", "json", "--no-interpolate", service}
	}
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir
	out, err := doExec(cmd)
	if err != nil {
		if strings.HasPrefix(err.Error(), "no such service") {
			return nil, nil
		}
		return nil, err
	}
	prj := ComposeProjectYaml{}
	err = json.NewDecoder(out).Decode(&prj)
	if err != nil {
		return nil, err
	}
	return prj.Services[service], nil
}

func CreateService(projectDir string, serviceName string, service map[string]interface{}) error {
	project, err := GetProject(projectDir, true)
	if err != nil {
		return err
	}
	project.Services[serviceName] = service
	projectName := project.Name
	if projectName == "" {
		projectName = path.Base(projectDir)
	}
	_, err = CreateProject(projectName, *project)
	if err != nil {
		return err
	}
	return nil
}
