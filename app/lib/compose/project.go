package compose

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	if err := login(); err != nil {
		panic(err)
	}

	// create project dirs if not exists
	if _, err := os.Stat(PROJECT_DIRS); os.IsNotExist(err) {
		if err := os.MkdirAll(PROJECT_DIRS, 0755); err != nil {
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
	// make sure masked.env exists
	if _, err := os.Stat(path.Join(projectDir, "masked.env")); err != nil && os.IsNotExist(err) {
		ReadEnvFile(projectDir, true) // this will create masked.env if necessary
	}
	// go to project directory and trigger docker compose up
	args := []string{"--env-file", "masked.env", "config", "--format", "json", "--no-path-resolution"}
	if len(nointerpolate) > 0 && nointerpolate[0] {
		args = []string{"config", "--format", "json", "--no-interpolate", "--no-path-resolution"}
	}
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
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
	if len(nointerpolate) > 0 && nointerpolate[0] { // handle no interpolate volume problem
		interpolatedPrj, err := GetProject(projectDir)
		if err != nil {
			return nil, err
		}
		prj.Volumes = handleNonIntepolatedVolumeMap(prj.Volumes, interpolatedPrj.Volumes)
		for name, svc := range prj.Services {
			ivols, ok := svc["volumes"]
			if !ok {
				continue
			}
			interpolatedISvc, ok := interpolatedPrj.Services[name]
			if !ok {
				continue
			}
			interpolatedIVols, ok := interpolatedISvc["volumes"]
			if !ok {
				continue
			}
			switch vols := ivols.(type) {
			case map[string]interface{}:
				interpolatedVols, ok := interpolatedIVols.(map[string]interface{})
				if !ok {
					continue
				}
				vols = handleNonIntepolatedVolumeMap(vols, interpolatedVols)
				svc["volumes"] = vols
				prj.Services[name] = svc
			case []interface{}:
				interpolatedVols, ok := interpolatedIVols.([]interface{})
				if !ok {
					continue
				}
				vols = handleNonIntepolatedVolumeSlice(vols, interpolatedVols)
				svc["volumes"] = vols
				prj.Services[name] = svc
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

func login() error {
	// go to project directory and trigger docker compose up
	registryAuth := os.Getenv("REGISTRY_AUTHS")
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
