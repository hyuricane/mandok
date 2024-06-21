package middlewares

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func BasicAuth() echo.MiddlewareFunc {
	if os.Getenv("API_USERNAME") == "" || os.Getenv("API_PASSWORD") == "" {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}
	return middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == os.Getenv("API_USERNAME") && password == os.Getenv("API_PASSWORD") {
			return true, nil
		}
		return false, nil
	})
}
