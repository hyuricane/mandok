package web

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	apiHandlers "inovasiriset.co.id/docker/manager/web/handlers/api/v1"
	"inovasiriset.co.id/docker/manager/web/middlewares"
)

func ListenHttp() error {
	app := echo.New()
	app.HideBanner = true
	app.Use(middleware.Recover())
	app.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		c.Logger().Error(err)
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}
	app.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})
	apiHandlers.RouteDocker(app.Group("/api/docker", middlewares.BasicAuth()))
	apiHandlers.RouteDocker(app.Group("/api/docker", middlewares.BasicAuth()))
	apiHandlers.RouteRepo(app.Group("/api/repo", middlewares.BasicAuth()))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ip := os.Getenv("IP")
	if ip == "" {
		ip = "0.0.0.0"
	}

	return app.Start(ip + ":" + port)
}
