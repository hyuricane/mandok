package dashboard

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/conf"
	"inovasiriset.co.id/docker/manager/web/middlewares"
	"inovasiriset.co.id/docker/manager/web/templates/components"
	"inovasiriset.co.id/docker/manager/web/templates/pages"
)

func RouteDashboard(group *echo.Group) {
	group.GET("/login", login)
	group.POST("/login", doLogin)
	group.GET("/logout", logout)
	group.GET("/", dashboard, middlewares.CookieAuth())
	projectGroup := group.Group("/project")
	projectGroup.Use(middlewares.CookieAuth())
	projectGroup.GET("/:project", project)
	projectGroup.GET("/:project/start", startProject)
	projectGroup.GET("/:project/down", downProject)
	projectGroup.GET("/:project/edit", editProject)
	projectGroup.POST("/:project/edit", doEditProject)
	projectGroup.GET("/:project/service/:service/start", startService)
	projectGroup.GET("/:project/service/:service/restart", restartService)
	projectGroup.GET("/:project/service/:service/restart", pullStartService)
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
	project, err := compose.GetStatus(projectDir)
	if err != nil {
		log.Printf("[ERROR] compose get status (%s, %t) %v", name, true, err)
	}

	envVals, err := compose.ReadEnvFile(projectDir, true)
	if err != nil {
		log.Printf("[ERROR] compose read env file  (%s, %t) %v", name, true, err)
	}

	return pages.Project(name, project, envVals).Render(c.Request().Context(), c.Response().Writer)
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
	switch format {
	case "yaml":
		err := yaml.NewDecoder(bytes.NewBufferString(payload)).Decode(&newProject)
		if err != nil {
			return pages.EditProject(projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	case "json":
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

func restartService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	err := compose.StartProject(projectDir, true, false, serviceName)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
}

func pullStartService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	err := compose.StartProject(projectDir, false, true, serviceName)
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
	format := c.FormValue("format")
	if format == "" {
		format = "json"
	}
	payload := c.FormValue("content")
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

	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	secrets, err := compose.GetExistingSecretsEnvs(projectDir)
	if err != nil {
		return err
	}

	if name != "" {
		envVals = append(envVals, compose.EnvVal{
			Key: name,
			Val: value,
		})
	}
	if len(envs) > 0 {
		newEnvVals, err := compose.ReadEnvsFromBytes([]byte(envs))
		if err != nil {
			return err
		}
		envVals = append(envVals, newEnvVals...)
	}

	err = compose.WriteEnvFile(projectDir, envVals, secrets)
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
	err := compose.SetEnvSecret(projectDir, name, true)
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
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	exists := map[string]int{}
	for i, v := range envVals {
		exists[v.Key] = i
	}
	if i, ok := exists[name]; ok {
		envVals = append(envVals[:i], envVals[i+1:]...)
	}
	err = compose.WriteEnvFile(projectDir, envVals, nil)
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
	out, cancel, err := compose.LogStreamWithContext(c.Request().Context(), projectDir, serviceName, tail)
	if err != nil {
		return err
	}
	go func() {
		<-c.Request().Context().Done()
		cancel()
	}()
	components.StreamLog(out).Render(c.Request().Context(), c.Response().Writer)
	return nil
}

func login(c echo.Context) error {
	return pages.Login(c.QueryParam("username")).Render(c.Request().Context(), c.Response().Writer)
}

func logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
		MaxAge:   -1,
	})
	return c.Redirect(302, "/")
}

func doLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	if username != conf.AppConfig.APIUsername || password != conf.AppConfig.APIPassword {
		return c.Redirect(302, "/login?username="+username)
	}

	maxAge := 60 * 60 * 24 // 1 day

	token, err := middlewares.JwtGenerateToken(username, time.Duration(maxAge)*time.Second)
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		// TODO: check if we actually have SSL
		Secure: false, // conf.AppConfig.TraefikSSL,
	})
	return c.Redirect(302, "/")
}
