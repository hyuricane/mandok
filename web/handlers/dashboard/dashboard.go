package dashboard

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/web/handlers/templates/pages"
)

func RouteDashboard(group *echo.Group) {
	group.GET("/", dashboard)
	group.GET("/login", login)
	group.POST("/login", doLogin)
}

func dashboard(c echo.Context) error {
	return pages.Dashbaord().Render(c.Request().Context(), c.Response().Writer)
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
