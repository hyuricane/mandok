package compose

import (
	"context"
	"fmt"
	"path"
)

func GetService(projectDir string, service string, nointerpolate ...bool) (map[string]interface{}, error) {
	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return nil, err
	}

	configModel, err := project.ConfigModel(ctx)
	if err != nil {
		return nil, err
	}
	servicesConfigModel, ok := configModel["services"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no services found in project")
	}
	serviceModel, ok := servicesConfigModel[service].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no service %s found in project", service)
	}
	return serviceModel, nil
}

func CreateService(projectDir string, serviceName string, service map[string]interface{}) error {
	project, err := GetProject(projectDir, true)
	if err != nil {
		return err
	}
	project.Services[serviceName] = service
	projectName := path.Base(projectDir)
	if projectName == "" {
		projectName = project.Name
	}
	_, err = CreateProject(projectName, *project)
	if err != nil {
		return err
	}
	return nil
}
