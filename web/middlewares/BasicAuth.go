package middlewares

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"inovasiriset.co.id/docker/manager/conf"
)

func BasicAuth() echo.MiddlewareFunc {
	if conf.AppConfig.APIUsername == "" || conf.AppConfig.APIPassword == "" {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}
	return middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == conf.AppConfig.APIUsername && password == conf.AppConfig.APIPassword {
			return true, nil
		}
		return false, nil
	})
}
