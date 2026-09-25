package compose

import (
	"context"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/labstack/gommon/log"
)

type ImagePullRequest struct {
	ProjectDir string
	Service    string
	Image      string
}

var pullCh chan ImagePullRequest = make(chan ImagePullRequest, 64)
var upCh chan string = make(chan string, 64)
var upRequests = map[string][]api.UpOptions{}
var _composeTaskRunning = false

func addImagePullRequest(pr ImagePullRequest) {
	pullCh <- pr
	go processTasks()
}

func addStartProjectRequest(projectDir string, upOpt api.UpOptions) {
	if ur, ok := upRequests[projectDir]; !ok {
		upRequests[projectDir] = []api.UpOptions{
			upOpt,
		}
		upCh <- projectDir
	} else {
		ur = append(ur, upOpt)
	}
	go processTasks()
}

func isComposeTaskRunning() bool {
	return _composeTaskRunning
}

func processTasks() {
	if _composeTaskRunning {
		return
	}
	defer func() {
		_composeTaskRunning = false
	}()
	for {
		select {
		case pr := <-pullCh:
			if err := _handlePull(pr); err != nil {
				log.Error("[ERROR] image pull", pr)
			}
		case upr := <-upCh:
			if err := _handleUp(upr); err != nil {
				log.Error("[ERROR] up", upr, upRequests[upr])
			}
		}
	}
}

func _handlePull(pr ImagePullRequest) error {
	ctx := context.Background()
	err := PullImage(ctx, pr.Image)
	return err
}

func _handleUp(upr string) error {
	ctx := context.Background()
	urs, ok := upRequests[upr]
	if !ok {
		return nil
	}
	defer func() {
		delete(upRequests, upr)
		// TODO: we should handle memory leak for uncleaned gc
	}()

	project, err := LoadProject(ctx, upr)
	if err != nil {
		return err
	}
	apiClient := getAPI()
	ups, err := _mergeUpOptions(urs)
	if err != nil {
		return err
	}
	for _, up := range ups {
		err = apiClient.Up(ctx, project.Project, up)
		if err != nil {
			return err
		}
	}
	return nil
}

func _mergeUpOptions(upOpts []api.UpOptions) ([]api.UpOptions, error) {
	forces := []api.UpOptions{}
	diverges := []api.UpOptions{}
	nevers := []api.UpOptions{}
	for _, opt := range upOpts {
		switch opt.Create.Recreate {
		case api.RecreateForce:
			forces = append(forces, opt)
		case api.RecreateDiverged:
			diverges = append(diverges, opt)
		case api.RecreateNever:
			nevers = append(nevers, opt)
		}
	}
	retval := []api.UpOptions{}
	if len(forces) > 0 {
		opt := forces[0]
		services := []string{}
		for _, forceOpt := range forces {
			services = append(services, forceOpt.Start.Services...)
		}
		// remove duplicate services
		services = unique(services)
		opt.Start.Services = services
		opt.Create.Services = services
		retval = append(retval, opt)
	}
	if len(diverges) > 0 {
		opt := diverges[0]
		services := []string{}
		for _, forceOpt := range diverges {
			services = append(services, forceOpt.Start.Services...)
		}
		// remove duplicate services
		services = unique(services)
		opt.Start.Services = services
		opt.Create.Services = services

		retval = append(retval, opt)
	}
	if len(nevers) > 0 {
		opt := nevers[0]
		services := []string{}
		for _, neverOpt := range nevers {
			services = append(services, neverOpt.Start.Services...)
		}
		// remove duplicate services
		services = unique(services)
		opt.Start.Services = services
		opt.Create.Services = services
		retval = append(retval, opt)
	}
	return retval, nil
}

func unique[K comparable](arr []K) []K {
	seen := make(map[K]bool)
	result := []K{}
	for _, item := range arr {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
