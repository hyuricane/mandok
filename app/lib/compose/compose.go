package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	types "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/docker/docker/api/types/registry"
	dockerclient "github.com/docker/docker/client"
	"go.yaml.in/yaml/v3"
	"inovasiriset.co.id/docker/manager/conf"
)

var _composeApi *api.Compose
var _dockerClient *dockerclient.Client

const COMPOSE_VERSION = "2.40.3"

func init() {
	var err error
	_dockerClient, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	cli, err := command.NewDockerCli(command.WithAPIClient(_dockerClient))
	if err != nil {
		panic(err)
	}
	err = cli.Initialize(flags.NewClientOptions())
	if err != nil {
		panic(err)
	}
	go func() {
		for _, auth := range registryAuthsFromEnv() {
			log.Printf("[INFO] registry %s login", auth.ServerAddress)
			if ok, err := _dockerClient.RegistryLogin(context.Background(), auth); err != nil {
				log.Printf("[ERROR] registry %s login %v", auth.ServerAddress, err)
			} else {
				log.Printf("[INFO] registry %s login %v", auth.ServerAddress, ok)
			}
		}
	}()
	composeApi := compose.NewComposeService(cli)
	_composeApi = &composeApi
}

func mergeMap(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMap(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func getAPI() api.Compose {
	return *_composeApi
}

func registryAuthsFromEnv() []registry.AuthConfig {
	// username:password@registryhost,...
	registryAuth := conf.AppConfig.RegistryAuths
	registryAuths := strings.Split(registryAuth, ",")
	auths := []registry.AuthConfig{}
	for _, registryAuth := range registryAuths {
		registryAuth := strings.Split(registryAuth, "@")
		if len(registryAuth) != 2 {
			continue
		}
		auth := strings.Split(registryAuth[0], ":")
		if len(auth) != 2 {
			continue
		}
		auths = append(auths, registry.AuthConfig{
			Username:      auth[0],
			Password:      auth[1],
			ServerAddress: registryAuth[1],
		})
	}
	return auths
}

type Project struct {
	*types.Project
	option *cli.ProjectOptions
	model  map[string]interface{}
}

func (p *Project) ConfigModel(ctx context.Context) (map[string]interface{}, error) {
	if p.model != nil {
		return p.model, nil
	}
	models, err := p.option.LoadModel(ctx)
	if err != nil {
		return nil, err
	}
	p.model = models
	return models, nil
}

func (p *Project) ConfigModelYaml() ([]byte, error) {
	if p.model == nil {
		_, err := p.ConfigModel(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return yaml.Marshal(p.model)
}

func (p *Project) ConfigModelJson() ([]byte, error) {
	if p.model == nil {
		_, err := p.ConfigModel(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(p.model)
}

type LoadProjectOptions struct {
	NoInference bool
	ConfigFiles []string
}

func LoadProject(ctx context.Context, projectDir string, options ...LoadProjectOptions) (*Project, error) {
	projectDirAbs, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}
	candidates := []string{}
	if len(options) > 0 && len(options[0].ConfigFiles) > 0 {
		candidates = options[0].ConfigFiles
	}
	if len(candidates) == 0 {
		candidates = []string{"compose.yaml", "docker-compose.yml", "compose.yml"}
	}
	var cfgFile string
	for _, name := range candidates {
		p := filepath.Join(projectDirAbs, name)
		if _, err := os.Stat(p); err == nil {
			cfgFile = p
			break
		}
	}
	if cfgFile == "" {
		return nil, fmt.Errorf("no compose file found in %s", projectDirAbs)
	}
	projectOptionsFns := []cli.ProjectOptionsFn{
		cli.WithWorkingDirectory(projectDirAbs),
	}
	noInference := false
	if len(options) > 0 {
		noInference = options[0].NoInference
	}
	if !noInference {
		projectOptionsFns = append(projectOptionsFns,
			cli.WithEnvFiles(filepath.Join(projectDirAbs, ".env"), filepath.Join(projectDirAbs, "masked.env")),
			cli.WithDotEnv,
			cli.WithDefaultConfigPath,
			cli.WithInterpolation(true),
		)
	} else {
		projectOptionsFns = append(projectOptionsFns,
			cli.WithInterpolation(false),
			cli.WithResolvedPaths(false),
		)
	}
	projectOptions, err := cli.NewProjectOptions(
		[]string{cfgFile},
		projectOptionsFns...,
	)
	if err != nil {
		return nil, err
	}

	project, err := projectOptions.LoadProject(ctx)
	if err != nil {
		return nil, err
	}

	// apply missing docker compose labels to each services
	for name, service := range project.Services {
		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		if service.Labels["com.docker.compose.project"] == "" {
			if service.CustomLabels == nil {
				service.CustomLabels = make(map[string]string)
			}
			service.CustomLabels[api.OneoffLabel] = "False"
			service.CustomLabels[api.ProjectLabel] = project.Name
			service.CustomLabels[api.ConfigFilesLabel] = cfgFile
			service.CustomLabels[api.WorkingDirLabel] = projectDirAbs
			service.CustomLabels[api.ServiceLabel] = service.Name
			service.CustomLabels[api.VersionLabel] = COMPOSE_VERSION
			project.Services[name] = service
		}
	}
	return &Project{
		Project: project,
		option:  projectOptions,
	}, nil
}

func sanitizeProjectName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Trim(s, "-_")
	if s == "" {
		s = "project"
	}
	// Optional: ensure starts with letter
	if len(s) > 0 && !isLetter(rune(s[0])) {
		s = "p-" + s
	}
	// Max length ~50–100 chars depending on backend
	if len(s) > 68 {
		s = s[:68]
	}
	return s
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
