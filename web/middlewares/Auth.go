package middlewares

import "github.com/labstack/echo/v4"

var _basicAuthMiddleware = BasicAuth()
var _cookieAuthMiddleware = CookieAuth()

func MiddlewareAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, err := c.Cookie("token")
			if err != nil {
				return _basicAuthMiddleware(next)(c)
			}
			return _cookieAuthMiddleware(next)(c)
		}
	}
}
