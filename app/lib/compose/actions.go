package compose

import (
	"context"
	"fmt"
	"io"
	"net/url"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/image"
)

// type ImagePullProgress struct {
// 	ProjectDir string  `json:"projectDir"`
// 	Service    string  `json:"service"`
// 	Image      string  `json:"image"`
// 	Status     string  `json:"status"`  // "Downloading", "Extracting", "Complete"
// 	Current    int64   `json:"current"` // Total downloaded bytes across layers
// 	Total      int64   `json:"total"`   // Total image size across layers
// 	Percent    float64 `json:"percent"` // 0 - 100%
// 	Layer      string  `json:"layer"`   // Active layer ID
// }

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
	for _, service := range services {
		if needPull, _ := ImageNeedPull(ctx, project.Services[service].Image, pull); needPull {
			addImagePullRequest(ImagePullRequest{
				ProjectDir: projectDir,
				Service:    service,
				Image:      project.Services[service].Image,
			})
		}
	}
	addStartProjectRequest(projectDir, api.UpOptions{
		Create: api.CreateOptions{
			Recreate:      recreatePolicy,
			Services:      services,
			RemoveOrphans: true,
			AssumeYes:     true,
			Inherit:       true,
		},
		Start: api.StartOptions{
			Services: services,
			Project:  project.Project,
			Wait:     true,
		},
	})
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

func ImageNeedPull(ctx context.Context, imageName string, forcePull bool) (bool, error) {
	if forcePull {
		return true, nil
	}
	if imageName == "" {
		return false, nil
	}
	_, err := _dockerClient.ImageInspect(ctx, imageName)
	if err != nil && cerrdefs.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

// func PullImageWithProgress(ctx context.Context, projectDir string, imageName string, progressFunc func(ImagePullProgress)) error {
// 	out, err := _dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	defer out.Close()
// 	if progressFunc == nil {
// 		return nil
// 	}

// 	jm := jsonmessage.JSONMessage{}
// 	decoder := json.NewDecoder(out)
// 	layers := make(map[string]struct{ Current, Total int64 })
// 	for {
// 		if err := decoder.Decode(&jm); err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			return err
// 		}
// 		if jm.ID != "" && jm.Progress != nil {
// 			layers[jm.ID] = struct{ Current, Total int64 }{
// 				Current: jm.Progress.Current,
// 				Total:   jm.Progress.Total,
// 			}

// 			var sumCurrent, sumTotal int64
// 			for _, l := range layers {
// 				sumCurrent += l.Current
// 				sumTotal += l.Total
// 			}

// 			var percent float64
// 			if sumTotal > 0 {
// 				percent = float64(sumCurrent) / float64(sumTotal) * 100.0
// 			}

// 			progressFunc(ImagePullProgress{
// 				ProjectDir: projectDir,
// 				Image:      imageName,
// 				Status:     jm.Status,
// 				Current:    sumCurrent,
// 				Total:      sumTotal,
// 				Percent:    percent,
// 				Layer:      jm.ID,
// 			})
// 		}
// 	}
// 	return nil
// }

// func handleImagePullProgress(progress ImagePullProgress) {
// 	// TODO: push pull image progress to project sse event at
// }

func PullImage(ctx context.Context, ref string) error {
	var err error
	var reader io.ReadCloser
	// try no auth
	reader, err = _dockerClient.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		// if err is unauthorized try with auths
		imageUrl, urlErr := url.ParseRequestURI(ref)
		if urlErr != nil {
			return err
		}

		// try with auths
		if auths := registryAuthsFromEnv(imageUrl.Host); len(auths) > 0 {
			for _, auth := range auths {
				reader, err = _dockerClient.ImagePull(ctx, ref, image.PullOptions{
					RegistryAuth: fmt.Sprintf("%s:%s@%s", auth.Username, auth.Password, auth.ServerAddress),
				})
				if err == nil {
					defer reader.Close()
					return nil
				}
			}
		}
		if err != nil {
			return err
		}
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader) // consume stream until EOF
	return err
}
