package compose

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ComposeProjectYaml struct {
	Name     string                 `yaml:"-" json:"name,omitempty"`
	Version  string                 `yaml:"version,omitempty" json:"version,omitempty"`
	Services map[string]interface{} `yaml:"services" json:"services"`
	Volumes  map[string]interface{} `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks map[string]interface{} `yaml:"networks,omitempty" json:"networks,omitempty"`
}

type RepoAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Registry string `json:"registry"`
}

type ExpectedPSData struct {
	Service    string `json:"Service"`
	CreatedAt  string `json:"CreatedAt"`
	Image      string `json:"Image"`
	Status     string `json:"Status"`
	State      string `json:"State"`
	Size       string `json:"Size"`
	RunningFor string `json:"RunningFor"`
	ExitCode   int    `json:"ExitCode"`
}

type ExtendedPSData struct {
	ExpectedPSData `json:",inline"`
	ID             string `json:"ID"`
	Labels         string `json:"Labels"`
}

type ComposeError struct {
	s []string
}

func (err ComposeError) Error() string {
	return strings.Join(err.s, "\n")
}

func NewComposeError(buffErr *bytes.Buffer) *ComposeError {
	str := buffErr.String()
	if str == "" {
		return nil
	}
	strs := strings.Split(str, "\n")
	cleanstrs := []string{}
	for _, s := range strs {
		if s == "" {
			continue
		}
		if strings.Contains(s, "level=warning") {
			continue
		}
		cleanstrs = append(cleanstrs, s)
	}
	if len(cleanstrs) > 0 {
		return &ComposeError{s: cleanstrs}
	}
	return nil
}

var PROJECT_DIRS = "projects"
var NETWORK = "mandok"

func init() {
	if os.Getenv("PROJECT_DIRS") != "" {
		PROJECT_DIRS = os.Getenv("PROJECT_DIRS")
	}
	if os.Getenv("NETWORK") != "" {
		NETWORK = os.Getenv("NETWORK")
	}
}

func CreateProject(name string, projectConfig ComposeProjectYaml) (string, error) {
	if name == "" {
		return "", nil
	}

	if projectConfig.Services == nil {
		projectConfig.Services = map[string]interface{}{}
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
		return nil, err
	}
	prj := ComposeProjectYaml{}
	err = json.NewDecoder(out).Decode(&prj)
	if err != nil {
		return nil, err
	}
	return &prj, nil
}

func GetStatus(projectDir string, all bool, services ...string) (map[string]ExpectedPSData, error) {
	args := []string{"ps", "--format", "json"}
	if all {
		args = append(args, "-a")
	}
	args = append(args, services...)
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir

	out, err := doExec(cmd)
	if err != nil {
		return nil, err
	}
	outputStrs := strings.Split(out.String(), "\n")
	retval := map[string]ExpectedPSData{}
	for _, outputStr := range outputStrs {
		if outputStr == "" {
			continue
		}
		psData := ExpectedPSData{}
		if err := json.Unmarshal([]byte(outputStr), &psData); err != nil {
			return nil, err
		}
		retval[psData.Service] = psData
	}
	return retval, nil
}

func GetStatusExt(projectDir string, all bool, services ...string) (map[string]ExtendedPSData, error) {
	args := []string{"ps", "--format", "json"}
	if all {
		args = append(args, "-a")
	}
	args = append(args, services...)
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir

	out, err := doExec(cmd)
	if err != nil {
		return nil, err
	}
	outputStrs := strings.Split(out.String(), "\n")
	retval := map[string]ExtendedPSData{}
	for _, outputStr := range outputStrs {
		if outputStr == "" {
			continue
		}
		psData := ExtendedPSData{}
		if err := json.Unmarshal([]byte(outputStr), &psData); err != nil {
			return nil, err
		}
		retval[psData.Service] = psData
	}
	return retval, nil
}

func StartProject(projectDir string, restart bool, pull bool, services ...string) error {
	args := []string{"up", "-d"}
	if restart {
		args = append(args, "--force-recreate")
	}
	if pull {
		args = append(args, "--pull", "always")
	}
	args = append(args, services...)
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir
	_, err := doExec(cmd)
	if err != nil {
		return err
	}

	return nil
}

func StopProject(projectDir string, service ...string) error {
	args := []string{"stop"}
	args = append(args, service...)
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir

	_, err := doExec(cmd)

	if err != nil {
		if cErr := NewComposeError(bytes.NewBufferString(err.Error())); cErr != nil {
			return cErr
		}

		return err
	}
	return nil
}

func DownProject(projectDir string) error {
	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = projectDir
	_, err := doExec(cmd)

	if err != nil {
		if cErr := NewComposeError(bytes.NewBufferString(err.Error())); cErr != nil {
			return cErr
		}
		return err
	}
	return nil
}

func RouteService(projectDir string, serviceName string, domain string, port int) error {
	if projectDir == "" {
		return nil
	}
	if serviceName == "" {
		return nil
	}
	if domain == "" {
		return nil
	}

	project, err := GetProject(projectDir, true)
	if err != nil {
		return err
	}
	service, ok := project.Services[serviceName]
	if !ok {
		return errors.New("service not found")
	}
	serviceM, ok := service.(map[string]interface{})
	if !ok {
		return errors.New("service not found")
	}

	var labels map[string]interface{}
	if Ilabels, ok := serviceM["labels"]; !ok {
		labels = map[string]interface{}{}
	} else {
		labels, ok = Ilabels.(map[string]interface{})
		if !ok {
			return errors.New("internal server error")
		}
	}
	labels["traefik.enable"] = "true"
	labels["traefik.http.routers."+project.Name+"_"+serviceName+".rule"] = "Host(`" + domain + "`)"
	labels["traefik.docker.network"] = NETWORK
	if port != 0 {
		labels["traefik.http.services."+project.Name+"_"+serviceName+".loadbalancer.server.port"] = port
	} else {
		// delete label port
		delete(labels, "traefik.http.services."+project.Name+"_"+serviceName+".loadbalancer.server.port")
	}
	serviceM["labels"] = labels

	// attach to traefik network
	var networks map[string]interface{}
	networksI, ok := serviceM["networks"]
	if !ok {
		networks = map[string]interface{}{}
	} else {
		networks, ok = networksI.(map[string]interface{})
		if !ok {
			return errors.New("internal server error")
		}
	}
	networks[NETWORK] = nil
	serviceM["networks"] = networks
	project.Services[serviceName] = serviceM

	// attach traefik external network to project
	if project.Networks == nil {
		project.Networks = map[string]interface{}{}
	}
	project.Networks[NETWORK] = map[string]interface{}{
		"external": true,
	}

	_, err = CreateProject(project.Name, *project)
	if err != nil {
		return err
	}
	projectStatus, err := GetStatus(projectDir, false, serviceName)
	if err != nil {
		return err
	}
	if len(projectStatus) == 0 {
		return nil
	}
	if sstat, ok := projectStatus[serviceName]; ok && sstat.State == "running" {
		err = StartProject(projectDir, false, false, serviceName)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteRoute(projectDir string, serviceName string) error {
	if projectDir == "" {
		return nil
	}
	if serviceName == "" {
		return nil
	}

	project, err := GetProject(projectDir, true)
	if err != nil {
		return err
	}
	service, ok := project.Services[serviceName]
	if !ok {
		return errors.New("service not found")
	}
	serviceM, ok := service.(map[string]interface{})
	if !ok {
		return errors.New("service not found")
	}

	var labels map[string]interface{}
	if Ilabels, ok := serviceM["labels"]; !ok {
		labels = map[string]interface{}{}
	} else {
		labels, ok = Ilabels.(map[string]interface{})
		if !ok {
			return errors.New("internal server error")
		}
	}
	labels["traefik.enable"] = false
	serviceM["labels"] = labels

	project.Services[serviceName] = serviceM

	_, err = CreateProject(project.Name, *project)
	if err != nil {
		return err
	}
	projectStatus, err := GetStatus(projectDir, false, serviceName)
	if err != nil {
		return err
	}
	if len(projectStatus) == 0 {
		return nil
	}
	if sstat, ok := projectStatus[serviceName]; ok && sstat.State == "running" {
		err = StartProject(projectDir, false, false, serviceName)
		if err != nil {
			return err
		}
	}

	return nil
}

// dry run docker compose up
func TryProject(projectDir string, configFileName string) error {
	args := []string{"--dry-run"}
	if configFileName != "" {
		args = append(args, "-f", configFileName)
	}
	args = append(args, "up", "-d")
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir
	_, err := doExec(cmd)

	if err != nil {
		if cErr := NewComposeError(bytes.NewBufferString(err.Error())); cErr != nil {
			return cErr
		}
		return err
	}
	return nil
}

func ReadEnvFile(projectDir string, masked bool) (plain map[string]string, secret map[string]string, err error) {
	fileName := ".env"
	if masked {
		fileName = "masked.env"
	}
	payload, err := os.ReadFile(filepath.Join(projectDir, fileName))
	plain = map[string]string{}
	secret = map[string]string{}
	if err != nil {
		if os.IsNotExist(err) {
			return plain, secret, nil
		}
		return nil, nil, err
	}
	lines := bytes.Split(payload, []byte("\n"))
	isPlain := true
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if string(line) == "### secret" {
			isPlain = false
			continue
		}
		if isPlain {
			parts := bytes.Split(line, []byte("="))
			if len(parts) != 2 {
				continue
			}
			plain[string(parts[0])] = string(parts[1])
		} else {
			parts := bytes.Split(line, []byte("="))
			if len(parts) != 2 {
				continue
			}
			secret[string(parts[0])] = string(parts[1])
		}
	}
	return plain, secret, nil
}

func WriteEnvFile(projectDir string, plain, secret map[string]string) error {
	realBs := bytes.Buffer{}
	maskedBs := bytes.Buffer{}
	for k, v := range plain {
		realBs.WriteString(k)
		realBs.WriteString("=")
		realBs.WriteString(v)
		realBs.WriteString("\n")

		maskedBs.WriteString(k)
		maskedBs.WriteString("=")
		maskedBs.WriteString(v)
		maskedBs.WriteString("\n")
	}
	realBs.WriteString("\n### secret\n")
	maskedBs.WriteString("\n### secret\n")
	for k, v := range secret {
		realBs.WriteString(k)
		realBs.WriteString("=")
		realBs.WriteString(v)
		realBs.WriteString("\n")

		maskedBs.WriteString(k)
		maskedBs.WriteString("=******\n")
	}
	err := os.WriteFile(filepath.Join(projectDir, ".env"), realBs.Bytes(), 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(projectDir, "masked.env"), maskedBs.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

func AttachToDockerNetwork(networkName string, cid string) error {
	// attach container to network
	log.Printf("[DEBUG] attaching container %s to network %s", cid, networkName)
	networkExec := exec.Command("docker", "network", "connect", networkName, cid)
	_, err := doExec(networkExec)
	if err != nil {
		return err
	}
	return nil
}

func doExec(cmd *exec.Cmd) (*bytes.Buffer, error) {
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	err := cmd.Run()
	if err != nil {
		if buffErr.Len() > 0 {
			err = fmt.Errorf("%s", buffErr.String())
		}
		return nil, err
	}
	return buff, nil
}
