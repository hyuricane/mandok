package compose

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type ExpectedPSData struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
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

type ServiceStatus struct {
	Name     string `json:"Name"`
	State    string `json:"State"`
	Image    string `json:"Image"`
	Expected int    `json:"Expected"`
	Running  int    `json:"Running"`
	Route    string `json:"Route"`
}

func GetStatus(projectDir string, all bool, services ...string) (map[string]ServiceStatus, error) {
	retval := map[string]ServiceStatus{}
	prj, err := GetProject(projectDir, true)
	if err != nil {
		return nil, err
	}
	for k, v := range prj.Services {
		ss := ServiceStatus{
			Name: k,
		}
		if imageI, ok := v["image"]; ok {
			ss.Image = imageI.(string)
		}
		if labelsI, ok := v["labels"]; ok {
			var labels map[string]interface{}
			switch ls := labelsI.(type) {
			case map[string]interface{}:
				labels = ls
			case []string:
				labels = map[string]interface{}{}
				for _, l := range ls {
					parts := strings.Split(l, "=")
					if len(parts) != 2 {
						continue
					}
					labels[parts[0]] = parts[1]
				}
			default:
				continue
			}
			if len(labels) == 0 {
				log.Printf("[WARNING] labels is not map[string]interface{} %v", labelsI)
				continue
			}
			if route, ok := labels["traefik.http.routers."+prj.Name+"_"+k+".rule"]; ok {
				ss.Route = route.(string)
				ss.Route = strings.TrimPrefix(ss.Route, "Host(`")
				ss.Route = strings.TrimSuffix(ss.Route, "`)")
			}
		}

		if deployI, ok := v["deploy"]; ok {
			if deployM, ok := deployI.(map[string]interface{}); ok {
				if replicasI, ok := deployM["replicas"]; ok {
					switch replicas := replicasI.(type) {
					case int:
						ss.Expected = replicas
					case float64:
						ss.Expected = int(replicas)
					case string:
						ss.Expected, err = strconv.Atoi(replicas)
						if err != nil {
							ss.Expected = 1
						}
					}
				}
			}
		}
		retval[k] = ss
	}

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
	for _, outputStr := range outputStrs {
		if outputStr == "" {
			continue
		}
		psData := ExpectedPSData{}
		if err := json.Unmarshal([]byte(outputStr), &psData); err != nil {
			return nil, err
		}
		status, ok := retval[psData.Service]
		if ok {
			if status.Expected == 0 {
				status.Expected = 1
			}
			if status.State == "" {
				status.State = psData.State
			}
			if psData.State == "running" {
				status.Running++
				status.State = "running"
			}
			retval[psData.Service] = status
		}
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
