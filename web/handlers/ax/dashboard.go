package ax

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/conf"
	"inovasiriset.co.id/docker/manager/web/middlewares"
	axPages "inovasiriset.co.id/docker/manager/web/templates/axs/pages"
	"inovasiriset.co.id/docker/manager/web/templates/components"
)

func RouteDashboard(group *echo.Group) {
	group.GET("", dashboard, middlewares.CookieAuth("/ax/login"))
	group.GET("/login", login)
	group.POST("/login", doLogin)
	projectGroup := group.Group("/project")
	projectGroup.Use(middlewares.CookieAuth("/ax/login"))
	projectGroup.GET("/:project", project)
	projectGroup.GET("/:project/status", status)
	projectGroup.GET("/:project/start", startProject)
	projectGroup.GET("/:project/down", downProject)
	projectGroup.GET("/:project/edit", editProject)
	projectGroup.POST("/:project/edit", doEditProject)
	projectGroup.GET("/:project/events", getEvents)

	projectGroup.GET("/:project/service/:service/start", startService)
	projectGroup.GET("/:project/service/:service/stop", stopService)

	projectGroup.GET("/:project/service/:service/edit", editService)
	projectGroup.GET("/:project/service-new", addService)
	projectGroup.POST("/:project/service/:service", doEditService)
	projectGroup.GET("/:project/service/:service/log", getLog)
	projectGroup.POST("/:project/service", doEditService)

	projectGroup.GET("/:project/route/:service", editRoute)
	projectGroup.POST("/:project/route/:service", doEditRoute)

	projectGroup.GET("/:project/envs", envs)
	projectGroup.POST("/:project/envs", setEnv)
	projectGroup.GET("/:project/envs/secret/:name", setEnvSecret)
	projectGroup.GET("/:project/envs/delete/:name", deleteEnv)

	projectGroup.GET("-new", newProject)
	projectGroup.POST("-new", doNewProject)
}

func dashboard(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projects, err := compose.GetProjects()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] projects %v", projects)
	log.Printf("[DEBUG] renderLayout %t", renderLayout)
	return axPages.Dashboard(renderLayout, projects, nil).Render(c.Request().Context(), c.Response().Writer)
}

func project(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	name := c.Param("project")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
	}
	log.Printf("[DEBUG] project %v", name)
	log.Printf("[DEBUG] renderLayout %t", renderLayout)
	return axPages.Project(renderLayout, name, nil).Render(c.Request().Context(), c.Response().Writer)
}

func newProject(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	return axPages.NewProject(renderLayout, nil).Render(c.Request().Context(), c.Response().Writer)
}

func doNewProject(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.FormValue("name")
	config := compose.ComposeProjectYaml{}
	projectDir, err := compose.CreateProject(projectName, config)
	if err != nil {
		return axPages.NewProject(renderLayout, err).Render(c.Request().Context(), c.Response().Writer)
	}
	log.Printf("[DEBUG] new Project: %s", projectDir)
	return c.Redirect(302, "/ax/project/"+projectName)
}

func editProject(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.Param("project")
	format := c.QueryParam("format")
	if format != "yaml" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
	}
	project, err := compose.GetProject(projectDir, true)
	if err != nil {
		return axPages.EditProject(renderLayout, projectName, "", format, err).Render(c.Request().Context(), c.Response().Writer)
	}

	if project == nil {
		return axPages.EditProject(renderLayout, projectName, "", format, nil).Render(c.Request().Context(), c.Response().Writer)
	}
	var payload []byte
	switch format {
	case "yaml":
		payload, err = yaml.Marshal(project)
		if err != nil {
			return axPages.EditProject(renderLayout, projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	default:
		payload, err = json.MarshalIndent(project, "", "  ")
		if err != nil {
			return axPages.EditProject(renderLayout, projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	}
	return axPages.EditProject(renderLayout, projectName, string(payload), format, nil).Render(c.Request().Context(), c.Response().Writer)
}

func doEditProject(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax")
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
			return axPages.EditProject(renderLayout, projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	default:
		err := json.NewDecoder(bytes.NewBufferString(payload)).Decode(&newProject)
		if err != nil {
			return axPages.EditProject(renderLayout, projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
		}
	}
	_, err := compose.CreateProject(projectName, newProject)
	if err != nil {
		return axPages.EditProject(renderLayout, projectName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
	}
	return c.Redirect(302, "/ax/project/"+projectName)
}

func startProject(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/project/"+projectName)
	}
	err := compose.StartProject(projectDir, false, false)
	if err != nil {
		return c.Redirect(302, "/ax/project/"+projectName+"/status")
	}
	return c.Redirect(302, "/ax/project/"+projectName+"/status")
}

func downProject(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/project/")
	}
	err := compose.DownProject(projectDir)
	if err != nil {
		return c.Redirect(302, "/ax/project/"+projectName+"/status")
	}
	return c.Redirect(302, "/ax/project/"+projectName+"/status")
}

func startService(c echo.Context) error {
	isAlpine := c.Request().Header.Get("X-Alpine-Request") == "true"
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	isRestart := c.QueryParam("restart") == "true"
	isPull := c.QueryParam("pull") == "true"
	if projectDir == "" {
		if isAlpine {
			return c.Redirect(302, "/ax/project/"+projectName)
		}
		return c.Redirect(302, "/ax/project/"+projectName)
	}
	err := compose.StartProject(projectDir, isRestart, isPull, serviceName)
	if err != nil {
		if isAlpine {
			return c.Redirect(302, "/ax/project/"+projectName+"/status")
		}
		return c.Redirect(302, "/ax/project/"+projectName)
	}
	if isAlpine {
		return c.Redirect(302, "/ax/project/"+projectName+"/status")
	}
	return c.Redirect(302, "/ax/project/"+projectName)
}

func stopService(c echo.Context) error {
	isAlpine := c.Request().Header.Get("X-Alpine-Request") == "true"
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		if isAlpine {
			return c.Redirect(302, "/ax/project/"+projectName+"/status")
		}
		return c.Redirect(302, "/ax/project/"+projectName)
	}
	err := compose.StopProject(projectDir, serviceName)
	if err != nil {
		if isAlpine {
			return c.Redirect(302, "/ax/project/"+projectName+"/status")
		}
		return c.Redirect(302, "/ax/project/"+projectName)
	}
	if isAlpine {
		return c.Redirect(302, "/ax/project/"+projectName+"/status")
	}
	return c.Redirect(302, "/ax/project/"+projectName)
}

func editService(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.Param("project")
	serviceName := c.Param("service")
	format := c.QueryParam("format")
	if format == "" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax")
	}
	service, err := compose.GetService(projectDir, serviceName, true)
	if err != nil {
		return axPages.Service(renderLayout, projectName, serviceName, "", format, err).Render(c.Request().Context(), c.Response().Writer)
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
	return axPages.Service(renderLayout, projectName, serviceName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
}

func addService(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.Param("project")
	format := c.QueryParam("format")
	if format == "" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax")
	}
	return axPages.Service(renderLayout, projectName, "", "", format, nil).Render(c.Request().Context(), c.Response().Writer)
}

func doEditService(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.Param("project")
	serviceName := c.Param("service")
	if serviceName == "" {
		serviceName = c.FormValue("name")
	}
	format := c.FormValue("format")
	if format == "" {
		format = "json"
	}
	payload := c.FormValue(format)
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax")
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
		return axPages.Service(renderLayout, projectName, serviceName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
	}
	err = compose.CreateService(projectDir, serviceName, service)
	if err != nil {
		return axPages.Service(renderLayout, projectName, serviceName, string(payload), format, err).Render(c.Request().Context(), c.Response().Writer)
	}
	return c.Redirect(302, "/ax/project/"+projectName)
}

func editRoute(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax")
	}
	routes, err := compose.GetRoutes(projectDir, serviceName)
	if err != nil {
		return err
	}
	route, ok := routes[serviceName]
	if !ok {
		route = compose.ServiceRoute{}
	}
	return axPages.Route(renderLayout, projectName, serviceName, route, err).Render(c.Request().Context(), c.Response().Writer)
}

func doEditRoute(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
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
	return c.Redirect(302, "/ax/project/"+projectName)
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
		return c.Redirect(302, "/ax/")
	}
	out, cancel, err := compose.LogStreamWithContext(c.Request().Context(), projectDir, serviceName, tail)
	if err != nil {
		return err
	}
	go func() {
		<-c.Request().Context().Done()
		cancel()
	}()
	return components.StreamLog(out).Render(c.Request().Context(), c.Response().Writer)
}

func login(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	username := c.QueryParam("username")
	var err error = nil
	return axPages.Login(renderLayout, username, err).Render(c.Request().Context(), c.Response().Writer)
}

func logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	return c.Redirect(302, "/ax")
}

func doLogin(c echo.Context) error {
	renderLayout := c.Request().Header.Get("X-Alpine-Request") != "true"
	username := c.FormValue("username")
	password := c.FormValue("password")
	if username != conf.AppConfig.APIUsername || password != conf.AppConfig.APIPassword {
		return axPages.Login(renderLayout, username, errors.New("invalid username or password")).Render(c.Request().Context(), c.Response().Writer)
	}
	maxAge := 24 * 60 * 60
	token, err := middlewares.JwtGenerateToken(username, time.Second*time.Duration(maxAge))
	if err != nil {
		return axPages.Login(renderLayout, username, err).Render(c.Request().Context(), c.Response().Writer)
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
	return c.Redirect(302, "/ax")
}
