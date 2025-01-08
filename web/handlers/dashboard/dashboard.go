package dashboard

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/handlers/templates/pages"
	"inovasiriset.co.id/docker/manager/web/middlewares"
)

func RouteDashboard(group *echo.Group) {
	group.GET("/login", login)
	group.POST("/login", doLogin)
	group.GET("/", dashboard, middlewares.CookieAuth())
	projectGroup := group.Group("/project")
	projectGroup.Use(middlewares.CookieAuth())
	projectGroup.GET("/:project", project)
	projectGroup.GET("/:project/start", startProject)
	projectGroup.GET("/:project/down", downProject)
	projectGroup.GET("/:project/edit", editProject)
	projectGroup.POST("/:project/edit", doEditProject)
	projectGroup.GET("/:project/service/:service/start", startService)
	projectGroup.GET("/:project/service/:service/stop", stopService)
	projectGroup.GET("/:project/service/:service/edit", editService)
	projectGroup.GET("/:project/service-new", addService)
	projectGroup.POST("/:project/service/:service", doEditService)
	projectGroup.GET("/:project/service/:service/log", getLog)
	projectGroup.POST("/:project/service", doAddService)
	projectGroup.GET("/:project/route/:service", editRoute)
	projectGroup.POST("/:project/route/:service", doEditRoute)
	projectGroup.POST("/:project/env/plain", setEnv)
	projectGroup.POST("/:project/env/secret", setEnv)
	projectGroup.POST("/:project/env", setEnv)
	projectGroup.GET("/:project/env/secret/:name", setEnvSecret)
	projectGroup.GET("/:project/env/delete/:name", deleteEnv)
	projectGroup.GET("-new", newProject)
	projectGroup.POST("-new", doNewProject)
}

func dashboard(c echo.Context) error {
	projects, err := compose.GetProjects()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] projects %v", projects)
	return pages.Dashbaord(projects).Render(c.Request().Context(), c.Response().Writer)
}

func project(c echo.Context) error {
	name := c.Param("project")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	project, err := compose.GetStatus(projectDir, true)
	if err != nil {
		return err
	}

	plain, secret, err := compose.ReadEnvFile(projectDir, true)
	if err != nil {
		return err
	}

	return pages.Project(name, project, plain, secret).Render(c.Request().Context(), c.Response().Writer)
}

func newProject(c echo.Context) error {
	return pages.NewProject(nil).Render(c.Request().Context(), c.Response().Writer)
}

func doNewProject(c echo.Context) error {
	projectName := c.FormValue("name")
	config := compose.ComposeProjectYaml{}
	projectDir, err := compose.CreateProject(projectName, config)
	if err != nil {
		return pages.NewProject(err).Render(c.Request().Context(), c.Response().Writer)
	}
	log.Printf("[DEBUG] new Project: %s", projectDir)
	return c.Redirect(302, "/")
}

func editProject(c echo.Context) error {
	projectName := c.Param("project")
	format := c.QueryParam("format")
	if format != "yaml" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	project, err := compose.GetProject(projectDir, true)
	if err != nil {
		return err
	}

	if project == nil {
		return pages.EditProject(projectName, "", format, nil).Render(c.Request().Context(), c.Response().Writer)
	}
	var payload []byte
	if format == "yaml" {
		payload, err = yaml.Marshal(project)
		if err != nil {
			return err
		}
	} else {
		payload, err = json.MarshalIndent(project, "", "  ")
		if err != nil {
			return err
		}
	}
	return pages.EditProject(projectName, string(payload), format, nil).Render(c.Request().Context(), c.Response().Writer)
}

func doEditProject(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	format := c.FormValue("format")
	if format == "" {
		format = "json"
	}
	payload := c.FormValue(format)
	newProject := compose.ComposeProjectYaml{}
	if format == "yaml" {
		err := yaml.NewDecoder(bytes.NewBufferString(payload)).Decode(&newProject)
		if err != nil {
			return pages.EditProject(projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	} else if format == "json" {
		err := json.NewDecoder(bytes.NewBufferString(payload)).Decode(&newProject)
		if err != nil {
			return pages.EditProject(projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	}
	_, err := compose.CreateProject(projectName, newProject)
	if err != nil {
		return pages.EditProject(projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
	}
	return c.Redirect(302, "/project/"+projectName)
}

func startProject(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/project/"+projectName)
	}
	err := compose.StartProject(projectDir, false, false)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func downProject(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/project/"+projectName)
	}
	err := compose.DownProject(projectDir)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func startService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	err := compose.StartProject(projectDir, false, false, serviceName)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func stopService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	err := compose.StopProject(projectDir, serviceName)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

// func bindYamlFileInput(header *multipart.FileHeader, out interface{}) error {
// 	file, err := header.Open()
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()
// 	return yaml.NewDecoder(file).Decode(out)
// }

func editService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	format := c.QueryParam("format")
	if format == "" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	service, err := compose.GetService(projectDir, serviceName, true)
	if err != nil {
		return err
	}
	var payload []byte
	switch format {
	case "json":
		payload, err = json.MarshalIndent(service, "", "  ")
	case "yaml":
		payload, err = yaml.Marshal(service)
	default:
		return c.Redirect(302, c.Path())
	}
	if err != nil {
		return err
	}
	return pages.Service(projectName, serviceName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
}

func addService(c echo.Context) error {
	projectName := c.Param("project")
	format := c.QueryParam("format")
	if format == "" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	return pages.Service(projectName, "", "", format, nil).Render(c.Request().Context(), c.Response().Writer)
}

func doEditService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	format := c.FormValue("format")
	if format == "" {
		format = "json"
	}
	payload := c.FormValue(format)
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	service := map[string]interface{}{}
	var err error
	switch format {
	case "json":
		err = json.NewDecoder(bytes.NewBufferString(payload)).Decode(&service)
	case "yaml":
		err = yaml.NewDecoder(bytes.NewBufferString(payload)).Decode(&service)
	default:
		return c.Redirect(302, c.Path())
	}
	if err != nil {
		return pages.Service(projectName, serviceName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
	}
	err = compose.CreateService(projectDir, serviceName, service)
	if err != nil {
		return pages.Service(projectName, serviceName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
	}
	return c.Redirect(302, "/project/"+projectName)
}
func doAddService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.FormValue("name")
	serviceJson := c.FormValue("json")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	service := map[string]interface{}{}
	err := json.NewDecoder(bytes.NewBufferString(serviceJson)).Decode(&service)
	if err != nil {
		return err
	}
	err = compose.CreateService(projectDir, serviceName, service)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func editRoute(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	routes, err := compose.GetRoutes(projectDir, serviceName)
	if err != nil {
		return err
	}
	route, ok := routes[serviceName]
	if !ok {
		route = compose.ServiceRoute{}
	}
	return pages.Route(projectName, serviceName, route, err).Render(c.Request().Context(), c.Response().Writer)
}

func doEditRoute(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	route := compose.ServiceRoute{
		Domain: c.FormValue("domain"),
	}
	if c.FormValue("port") != "" {
		port, err := strconv.Atoi(c.FormValue("port"))
		if err != nil {
			return err
		}
		route.Port = port
	}
	stickyStr := c.FormValue("sticky")
	sticky := map[string]interface{}{}
	if stickyStr != "" {
		err := json.NewDecoder(bytes.NewBufferString(stickyStr)).Decode(&sticky)
		if err != nil {
			return err
		}
	}
	route.Sticky = sticky
	err := compose.RouteService(projectDir, serviceName, route)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func setEnv(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	name := c.FormValue("name")
	value := c.FormValue("value")
	envs := c.FormValue("envs")
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	if envs != "" {
		newPlains := map[string]string{}
		lines := strings.Split(envs, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				continue
			}
			if line == "" {
				continue
			}
			linec := strings.SplitN(line, "=", 2)
			if len(linec) == 2 {
				newPlains[linec[0]] = linec[1]
			}
		}
		for k, v := range newPlains {
			plain[k] = v
			delete(secret, k)
		}
	}
	if name != "" {
		if path.Base(c.Path()) == "plain" {
			plain[name] = value
			delete(secret, name)
		} else {
			secret[name] = value
			delete(plain, name)
		}
	}

	err = compose.WriteEnvFile(projectDir, plain, secret)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func setEnvSecret(c echo.Context) error {
	projectName := c.Param("project")
	name := c.Param("name")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	if v, ok := plain[name]; ok {
		secret[name] = v
		delete(plain, name)
	}
	err = compose.WriteEnvFile(projectDir, plain, secret)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func deleteEnv(c echo.Context) error {
	projectName := c.Param("project")
	name := c.Param("name")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	delete(plain, name)
	delete(secret, name)
	err = compose.WriteEnvFile(projectDir, plain, secret)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func getLog(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	tail := 10
	if c.QueryParam("tail") != "" {
		tail, _ = strconv.Atoi(c.QueryParam("tail"))
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	out, cancel, err := compose.LogStream(projectDir, serviceName, tail)
	if err != nil {
		return err
	}
	defer cancel()
	pages.StreamLog(out).Render(c.Request().Context(), c.Response().Writer)
	return nil
}

func login(c echo.Context) error {
	return pages.Login(c.QueryParam("username")).Render(c.Request().Context(), c.Response().Writer)
}

func doLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	if username != os.Getenv("API_USERNAME") || password != os.Getenv("API_PASSWORD") {
		return c.Redirect(302, "/login?username="+username)
	}

	token64 := bytes.NewBuffer([]byte{})
	enc := base64.NewEncoder(base64.StdEncoding, token64)
	_, err := enc.Write([]byte(username + ":" + password))
	if err != nil {
		return err
	}
	err = enc.Close()
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:  "token",
		Value: token64.String(),
		Path:  "/",
	})
	return c.Redirect(302, "/")
}
