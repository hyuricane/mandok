package web

import (
	"io/fs"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	apiHandlers "inovasiriset.co.id/docker/manager/web/handlers/api/v1"
	"inovasiriset.co.id/docker/manager/web/handlers/ax"
	"inovasiriset.co.id/docker/manager/web/handlers/dashboard"
	"inovasiriset.co.id/docker/manager/web/handlers/hx"
	"inovasiriset.co.id/docker/manager/web/middlewares"
)

type FallingBackFS struct {
	FSs []fs.FS
}

func (f *FallingBackFS) Open(name string) (fs.File, error) {
	for _, fsys := range f.FSs {
		file, err := fsys.Open(name)
		if err == nil {
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}
func ListenHttp(statics map[string][]fs.FS) error {
	app := echo.New()
	app.HideBanner = true
	app.Pre(middleware.RemoveTrailingSlash())
	app.Use(middleware.Recover())
	app.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		c.Logger().Error(err, c.Path())
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}
	for prefix, fsys := range statics {
		if fsys == nil {
			continue
		}
		if len(fsys) == 0 {
			continue
		}
		if len(fsys) == 1 {
			app.StaticFS(prefix, fsys[0])
		} else {
			app.StaticFS(prefix, &FallingBackFS{FSs: fsys})
		}
	}
	dashboard.RouteDashboard(app.Group(""))
	hx.RouteDashboard(app.Group("/hx"))
	ax.RouteDashboard(app.Group("/ax"))

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
