package web

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	apiHandlers "inovasiriset.co.id/docker/manager/web/handlers/api/v1"
	"inovasiriset.co.id/docker/manager/web/handlers/dashboard"
	"inovasiriset.co.id/docker/manager/web/handlers/hx"
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
	app.Static("/static", "./static")
	dashboard.RouteDashboard(app.Group(""))
	hx.RouteDashboard(app.Group("/hx"))

	apiHandlers.RouteDocker(app.Group("/api/docker", middlewares.MiddlewareAuth()))
	apiHandlers.RouteCompose(app.Group("/api/compose", middlewares.MiddlewareAuth()))
	apiHandlers.RouteRepo(app.Group("/api/repo", middlewares.MiddlewareAuth()))

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
