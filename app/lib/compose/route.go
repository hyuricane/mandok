package compose

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type ServiceRoute struct {
	Domain string                 `json:"domain"`
	Port   int                    `json:"port,omitempty"`
	Sticky map[string]interface{} `json:"sticky,omitempty"`
}

var NETWORK = "mandok"

var TRAEFIK = false
var TRAEFIK_SSL = false
var TRAEFIK_HTTP_ENTRYPOINT = "web"
var TRAEFIK_HTTPS_ENTRYPOINT = "websecure"

func init() {
	if os.Getenv("NETWORK") != "" {
		NETWORK = os.Getenv("NETWORK")
	}
	if traefikStr := os.Getenv("TRAEFIK"); traefikStr != "" {
		TRAEFIK, _ = strconv.ParseBool(traefikStr)
	}
	if traefikSSLStr := os.Getenv("TRAEFIK_SSL"); traefikSSLStr != "" {
		TRAEFIK_SSL, _ = strconv.ParseBool(traefikSSLStr)
	}
	if traefikHTTPSEntrypoint := os.Getenv("TRAEFIK_HTTP_ENTRYPOINT"); traefikHTTPSEntrypoint != "" {
		TRAEFIK_HTTP_ENTRYPOINT = traefikHTTPSEntrypoint
	}
	if traefikHTTPSEntrypoint := os.Getenv("TRAEFIK_HTTPS_ENTRYPOINT"); traefikHTTPSEntrypoint != "" {
		TRAEFIK_HTTPS_ENTRYPOINT = traefikHTTPSEntrypoint
	}
}

func RouteService(projectDir string, serviceName string, route ServiceRoute) error {
	if !TRAEFIK {
		return nil
	}
	if projectDir == "" {
		return nil
	}
	if serviceName == "" {
		return nil
	}
	if route.Domain == "" {
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

	var labels map[string]interface{}
	if Ilabels, ok := service["labels"]; !ok {
		labels = map[string]interface{}{}
	} else {
		labels, ok = Ilabels.(map[string]interface{})
		if !ok {
			return errors.New("internal server error")
		}
	}
	labels["traefik.enable"] = "true"
	labels["traefik.docker.network"] = NETWORK
	// port
	if route.Port != 0 {
		labels["traefik.http.services."+project.Name+"_"+serviceName+".loadbalancer.server.port"] = route.Port
	} else {
		// delete label port
		delete(labels, "traefik.http.services."+project.Name+"_"+serviceName+".loadbalancer.server.port")
	}
	// sticky
	for k, v := range route.Sticky {
		stickyName := k
		switch s := v.(type) {
		case map[string]interface{}:
			for k, v := range s {
				labels[fmt.Sprintf("traefik.http.services.%s_%s.loadbalancer.sticky.%s.%s", project.Name, serviceName, stickyName, k)] = v
			}
		default:
			labels[fmt.Sprintf("traefik.http.services.%s_%s.loadbalancer.sticky.%s", project.Name, serviceName, stickyName)] = v
		}
	}

	// http
	labels["traefik.http.routers."+project.Name+"_"+serviceName+".rule"] = "Host(`" + route.Domain + "`)"
	labels["traefik.http.routers."+project.Name+"_"+serviceName+".entrypoints"] = TRAEFIK_HTTP_ENTRYPOINT

	if TRAEFIK_SSL {
		labels["traefik.http.routers."+project.Name+"_"+serviceName+"-secure.rule"] = "Host(`" + route.Domain + "`)"
		labels["traefik.http.routers."+project.Name+"_"+serviceName+"-secure.entrypoints"] = TRAEFIK_HTTPS_ENTRYPOINT
		labels["traefik.http.routers."+project.Name+"_"+serviceName+"-secure.tls"] = true

		// redirect to https
		labels["traefik.http.routers."+project.Name+"_"+serviceName+".middlewares"] = "https_redirect"
		labels["traefik.http.middlewares.https_redirect.redirectscheme.scheme"] = "https"
	}

	service["labels"] = labels

	// attach to traefik network
	var networks map[string]interface{}
	networksI, ok := service["networks"]
	if !ok {
		networks = map[string]interface{}{}
	} else {
		networks, ok = networksI.(map[string]interface{})
		if !ok {
			return errors.New("internal server error")
		}
	}
	networks[NETWORK] = nil
	service["networks"] = networks
	project.Services[serviceName] = service

	// attach traefik external network to project
	if project.Networks == nil {
		project.Networks = map[string]interface{}{}
	}
	project.Networks[NETWORK] = map[string]interface{}{
		"external": true,
	}

	projectName := path.Base(projectDir)
	if projectName == "" {
		projectName = project.Name
	}
	_, err = CreateProject(projectName, *project)
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

	var labels map[string]interface{}
	if Ilabels, ok := service["labels"]; !ok {
		labels = map[string]interface{}{}
	} else {
		labels, ok = Ilabels.(map[string]interface{})
		if !ok {
			return errors.New("internal server error")
		}
	}
	labels["traefik.enable"] = false
	service["labels"] = labels

	project.Services[serviceName] = service

	projectName := path.Base(projectDir)
	if projectName == "" {
		projectName = project.Name
	}
	_, err = CreateProject(projectName, *project)
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

func GetRoutes(projectDir string, services ...string) (map[string]ServiceRoute, error) {
	if projectDir == "" {
		return nil, nil
	}
	args := []string{"config", "--format", "json", "--no-path-resolution"}
	args = append(args, services...)

	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = projectDir
	out, err := doExec(cmd)
	if err != nil {
		return nil, err
	}
	project := ComposeProjectYaml{}
	err = json.NewDecoder(out).Decode(&project)
	if err != nil {
		return nil, err
	}

	serviceRoutes := map[string]ServiceRoute{}
	for serviceName, serviceConfig := range project.Services {
		labels, ok := serviceConfig["labels"]
		if !ok {
			continue
		}
		labelsMap, ok := labels.(map[string]interface{})
		if !ok {
			continue
		}
		if enabled, ok := labelsMap["traefik.enable"]; !ok || enabled != "true" {
			continue
		}
		serviceRoute := ServiceRoute{}
		for k, v := range labelsMap {
			if strings.HasPrefix(k, "traefik.http.") {
				switch k {
				case fmt.Sprintf("traefik.http.routers.%s_%s.rule", project.Name, serviceName):
					domain := strings.TrimPrefix(v.(string), "Host(`")
					domain = strings.TrimSuffix(domain, "`)")
					serviceRoute.Domain = domain
				case fmt.Sprintf("traefik.http.services.%s_%s.loadbalancer.server.port", project.Name, serviceName):
					if str, ok := v.(string); ok {
						port, err := strconv.Atoi(str)
						if err != nil {
							return nil, err
						}
						serviceRoute.Port = port
					} else if i, ok := v.(int); ok {
						serviceRoute.Port = i
					}
				default:
					pref := fmt.Sprintf("traefik.http.services.%s_%s.loadbalancer.sticky.", project.Name, serviceName)
					if strings.HasPrefix(k, pref) {
						if serviceRoute.Sticky == nil {
							serviceRoute.Sticky = map[string]interface{}{}
						}
						// process sticky
						k = strings.TrimPrefix(k, pref)
						stickies := strings.Split(k, ".")
						if len(stickies) < 2 {
							continue
						}
						mI, ok := serviceRoute.Sticky[stickies[0]]
						if !ok {
							mI = map[string]interface{}{}
						}
						m, ok := mI.(map[string]interface{})
						if !ok {
							continue
						}
						m[stickies[1]] = v
						serviceRoute.Sticky[stickies[0]] = m
					}
				}
			}
		}
		serviceRoutes[serviceName] = serviceRoute
	}
	return serviceRoutes, nil
}
