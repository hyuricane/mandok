package compose

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/docker/compose/v2/pkg/api"
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
	Route    string `json:"Route,omitempty"`
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

	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return nil, err
	}
	psAll := len(services) == 0
	apiClient := getAPI()
	psSummary, err := apiClient.Ps(ctx, project.Name, api.PsOptions{
		Project:  project.Project,
		Services: services,
		All:      psAll,
	})
	if err != nil {
		return nil, err
	}
	for _, ps := range psSummary {
		status, ok := retval[ps.Service]
		if ok {
			if status.Expected == 0 {
				status.Expected = 1
			}
			if status.State == "" {
				status.State = ps.State
			}
			if ps.State == "running" {
				status.Running++
				status.State = "running"
			}
			retval[ps.Service] = status
		}
	}

	return retval, nil
}

func GetStatusExt(projectDir string, all bool, services ...string) (map[string]ExtendedPSData, error) {
	project, err := LoadProject(context.Background(), projectDir)
	if err != nil {
		return nil, err
	}

	apiClient := getAPI()
	psSummary, err := apiClient.Ps(context.Background(), project.Name, api.PsOptions{
		Project:  project.Project,
		Services: services,
		All:      all,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	retval := map[string]ExtendedPSData{}
	for _, ps := range psSummary {
		createdAt := time.Unix(ps.Created, 0)

		var runningFor string
		switch ps.State {
		case "running":
			duration := now.Sub(createdAt).Round(time.Second)
			runningFor = fmt.Sprintf("Up %s", formatDurationHuman(duration))
		case "exited", "dead":
			duration := now.Sub(createdAt).Round(time.Second)
			runningFor = fmt.Sprintf("Exited (%d) %s ago", ps.ExitCode, formatDurationHuman(duration))
		default:
			runningFor = ps.Status // fallback to raw status if weird state
		}
		var labels string
		for k, v := range ps.Labels {
			labels += fmt.Sprintf("%s=%s,", k, v)
		}
		retval[ps.Service] = ExtendedPSData{
			ExpectedPSData: ExpectedPSData{
				ID:         ps.ID,
				Name:       ps.Name,
				Service:    ps.Service,
				CreatedAt:  createdAt.Format(time.DateTime),
				Image:      ps.Image,
				Status:     ps.Status,
				State:      ps.State,
				Size:       "0B (not available)",
				RunningFor: runningFor,
				ExitCode:   ps.ExitCode,
			},
			ID:     ps.ID,
			Labels: labels,
		}

	}
	return retval, nil
}

func formatDurationHuman(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "about an hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
