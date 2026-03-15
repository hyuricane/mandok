package middlewares

import (
	"github.com/labstack/echo/v4"
)

func MiddlewareAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, err := c.Cookie("token")
			if err != nil {
				return BasicAuth()(next)(c)
			}
			return CookieAuth()(next)(c)
		}
	}
}
