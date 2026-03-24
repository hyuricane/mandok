package compose

import (
	"bytes"
	"context"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"inovasiriset.co.id/docker/manager/conf"
)

func init() {
	if err := login(); err != nil {
		panic(err)
	}

	// create project dirs if not exists
	if _, err := os.Stat(conf.AppConfig.ProjectDirs); os.IsNotExist(err) {
		if err := os.MkdirAll(conf.AppConfig.ProjectDirs, 0755); err != nil {
			log.Fatalf("[ERROR] create project dirs %v", err)
		}
	}
}

type ComposeProjectYaml struct {
	Name     string                            `yaml:"-" json:"name,omitempty"`
	Version  string                            `yaml:"version,omitempty" json:"version,omitempty"`
	Services map[string]map[string]interface{} `yaml:"services" json:"services"`
	Volumes  map[string]interface{}            `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks map[string]interface{}            `yaml:"networks,omitempty" json:"networks,omitempty"`
	Configs  map[string]interface{}            `yaml:"configs,omitempty" json:"configs,omitempty"`
}

func CreateProject(name string, projectConfig ComposeProjectYaml) (string, error) {
	if name == "" {
		return "", nil
	}

	if projectConfig.Services == nil {
		projectConfig.Services = map[string]map[string]interface{}{}
	}
	// create dir
	projectPath := path.Join(conf.AppConfig.ProjectDirs, name)
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
	projectPath := path.Join(conf.AppConfig.ProjectDirs, name)
	projectDir := filepath.Join(conf.AppConfig.ProjectDirs, name)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return ""
	} else if err != nil {
		log.Printf("[ERROR] hasProject %v", err)
		return ""
	}
	return projectPath
}

func GetProject(projectDir string, nointerpolate ...bool) (*ComposeProjectYaml, error) {
	// make sure masked.env exists
	_, err := ReadEnvFile(projectDir, true)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir, LoadProjectOptions{
		NoInference: len(nointerpolate) > 0 && nointerpolate[0],
	})
	if err != nil {
		return nil, err
	}

	configModel, err := project.ConfigModel(ctx)
	if err != nil {
		return nil, err
	}
	prj := ComposeProjectYaml{}
	prj.Name = project.Name
	prj.Version, _ = configModel["version"].(string)
	prj.Volumes, _ = configModel["volumes"].(map[string]interface{})
	prj.Networks, _ = configModel["networks"].(map[string]interface{})
	prj.Configs, _ = configModel["configs"].(map[string]interface{})
	if svcs, ok := configModel["services"].(map[string]interface{}); ok {
		prj.Services = make(map[string]map[string]interface{})
		for name, svc := range svcs {
			if svcMap, ok := svc.(map[string]interface{}); ok {
				prj.Services[name] = svcMap
			}
		}
	}
	return &prj, nil
}

func handleNonIntepolatedVolumeMap(nonInterpolatedVolumes, interpolatedVolumes map[string]interface{}) map[string]interface{} {
	for name, ivol := range nonInterpolatedVolumes {
		vol, ok := ivol.(map[string]interface{})
		if !ok {
			continue
		}
		t, ok := vol["type"]
		if !ok {
			continue
		}
		ts, ok := t.(string)
		if !ok {
			continue
		}
		if ts != "volume" {
			continue
		}
		interpoatedIvol, ok := interpolatedVolumes[name]
		if !ok {
			continue
		}
		interpolatedVol, ok := interpoatedIvol.(map[string]interface{})
		if !ok {
			continue
		}
		interpolatedIType, ok := interpolatedVol["type"]
		if !ok {
			continue
		}
		interpolatedType, ok := interpolatedIType.(string)
		if !ok {
			continue
		}
		if interpolatedType == "bind" {
			vol["type"] = "bind"
			vol["bind"] = interpolatedVol["bind"]
			delete(vol, ts)
			nonInterpolatedVolumes[name] = vol
		}
	}
	return nonInterpolatedVolumes
}

func handleNonIntepolatedVolumeSlice(nonInterpolatedVolumes, interpolatedVolumes []interface{}) []interface{} {
	for i, ivol := range nonInterpolatedVolumes {
		vol, ok := ivol.(map[string]interface{})
		if !ok {
			continue
		}
		t, ok := vol["type"]
		if !ok {
			continue
		}
		ts, ok := t.(string)
		if !ok {
			continue
		}
		if ts != "volume" {
			continue
		}
		// len([0]) = 1
		if len(interpolatedVolumes) <= i {
			continue
		}

		interpolatedIVol := interpolatedVolumes[i]
		if !ok {
			continue
		}
		interpolatedVol, ok := interpolatedIVol.(map[string]interface{})
		if !ok {
			continue
		}
		interpolatedIType, ok := interpolatedVol["type"]
		if !ok {
			continue
		}
		interpolatedType, ok := interpolatedIType.(string)
		if !ok {
			continue
		}
		if interpolatedType == "bind" {
			vol["type"] = "bind"
			vol["bind"] = interpolatedVol["bind"]
			delete(vol, ts)
			nonInterpolatedVolumes[i] = vol
		}
	}
	return nonInterpolatedVolumes
}

func GetProjects() ([]string, error) {
	projects := []string{}
	files, err := os.ReadDir(conf.AppConfig.ProjectDirs)
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

func login() error {
	// go to project directory and trigger docker compose up
	registryAuth := conf.AppConfig.RegistryAuths
	if registryAuth == "" {
		return nil
	}

	parsed, err := url.Parse("http://" + registryAuth)
	if err != nil {
		return err
	}
	args := []string{
		"login",
	}
	password := ""
	if parsed.User != nil {
		args = append(args, "-u "+parsed.User.Username())
		if pass, ok := parsed.User.Password(); ok {
			password = pass
			args = append(args, "--password-stdin")
		}
	}

	args = append(args, parsed.Host)
	cmd := exec.Command("docker", args...)
	if password != "" {
		cmd.Stdin = strings.NewReader(password)
	}
	out, err := doExec(cmd)
	if err != nil {
		return NewComposeError(bytes.NewBufferString(err.Error()))
	}
	if out == nil {
		return nil
	}
	return nil
}
