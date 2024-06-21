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
	app.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})
	apiHandlers.RouteDocker(app.Group("/api/docker", middlewares.BasicAuth()))

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
