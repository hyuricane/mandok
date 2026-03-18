package compose

import (
	"context"

	"github.com/docker/compose/v2/pkg/api"
)

func StartProject(projectDir string, forceRecreate bool, pull bool, services ...string) error {
	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return err
	}

	if len(services) > 0 {
		project.Project, err = project.Project.WithSelectedServices(services)
		if err != nil {
			return err
		}
	}

	recreatePolicy := api.RecreateDiverged
	if forceRecreate {
		recreatePolicy = api.RecreateForce
	}

	// apiClient is github.com/docker/compose/v2/pkg/api Compose
	apiClient := getAPI()
	err = apiClient.Up(ctx, project.Project, api.UpOptions{
		Create: api.CreateOptions{
			Build: &api.BuildOptions{
				Pull: pull,
			},
			Recreate:      recreatePolicy,
			Services:      services,
			RemoveOrphans: true,
			AssumeYes:     true,
		},
		Start: api.StartOptions{
			Services: services,
			Project:  project.Project,
			Wait:     true,
		},
	})
	if err != nil {
		return err
	}

	// log.Printf("[DEBUG] project model %v", )

	return nil
}

func StopProject(projectDir string, service ...string) error {
	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return err
	}
	apiClient := getAPI()
	err = apiClient.Stop(ctx, project.Name, api.StopOptions{
		Project:  project.Project,
		Services: service,
	})
	if err != nil {
		return err
	}
	return nil
}

func DownProject(projectDir string) error {
	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return err
	}
	apiClient := getAPI()
	err = apiClient.Down(ctx, project.Name, api.DownOptions{
		Project: project.Project,
	})
	if err != nil {
		return err
	}
	return nil
}
