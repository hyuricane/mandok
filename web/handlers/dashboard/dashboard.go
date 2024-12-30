package dashboard

import (
	"bytes"
	"encoding/base64"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/handlers/templates/pages"
	"inovasiriset.co.id/docker/manager/web/middlewares"
)

func RouteDashboard(group *echo.Group) {
	group.GET("/login", login)
	group.POST("/login", doLogin)
	group.GET("/", dashboard, middlewares.CookieAuth())
	group.GET("/:project", project, middlewares.CookieAuth())
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

	routes, err := compose.GetRoutes(projectDir)
	if err != nil {
		return err
	}

	return pages.Project(name, project, routes).Render(c.Request().Context(), c.Response().Writer)
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
