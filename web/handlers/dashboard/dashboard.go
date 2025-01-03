package dashboard

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/labstack/echo/v4"
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
	projectGroup.GET("/:project/service/:service/start", startService)
	projectGroup.GET("/:project/service/:service/stop", stopService)
	projectGroup.GET("/:project/service/:service/edit", editService)
	projectGroup.GET("/:project/service-new", addService)
	projectGroup.POST("/:project/service/:service", doEditService)
	projectGroup.POST("/:project/service", doAddService)
	projectGroup.GET("/:project/route/:service", editRoute)
	projectGroup.POST("/:project/route/:service", doEditRoute)
	projectGroup.POST("/:project/env/plain", setEnv)
	projectGroup.POST("/:project/env/secret", setEnv)
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
	content := c.FormValue("json")
	buff := bytes.NewBuffer([]byte(content))
	config := compose.ComposeProjectYaml{}
	err := json.NewDecoder(buff).Decode(&config)
	if err != nil {
		return pages.NewProject(err).Render(c.Request().Context(), c.Response().Writer)
	}
	projectDir, err := compose.CreateProject(c.FormValue("name"), config)
	if err != nil {
		return pages.NewProject(err).Render(c.Request().Context(), c.Response().Writer)
	}
	log.Printf("[DEBUG] new Project: %s", projectDir)
	return c.Redirect(302, "/")
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

func editService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	service, err := compose.GetService(projectDir, serviceName, true)
	if err != nil {
		return err
	}
	buff := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(buff)
	enc.SetIndent("", "  ")
	err = enc.Encode(service)
	if err != nil {
		return err
	}
	return pages.Service(projectName, serviceName, buff.String(), err).Render(c.Request().Context(), c.Response().Writer)
}

func addService(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	return pages.Service(projectName, "", "", nil).Render(c.Request().Context(), c.Response().Writer)
}

func doEditService(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
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
	if len(routes) == 0 {
		return c.Redirect(302, "/project/"+projectName)
	}
	return pages.Route(projectName, serviceName, routes[serviceName], err).Render(c.Request().Context(), c.Response().Writer)
}

func doEditRoute(c echo.Context) error {
	projectName := c.Param("project")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	route := compose.ServiceRoute{}
	err := json.NewDecoder(bytes.NewBufferString(c.FormValue("json"))).Decode(&route)
	if err != nil {
		return err
	}
	err = compose.RouteService(projectDir, serviceName, route)
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
	if projectDir == "" {
		return c.Redirect(302, "/")
	}
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	if path.Base(c.Path()) == "plain" {
		plain[name] = value
	} else {
		secret[name] = value
	}

	err = compose.WriteEnvFile(projectDir, plain, secret)
	if err != nil {
		return err
	}
	return c.Redirect(302, "/project/"+projectName)
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
