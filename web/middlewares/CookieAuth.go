package middlewares

import (
	"bytes"
	"encoding/base64"
	"os"

	"github.com/labstack/echo/v4"
)

func CookieAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("token")
			if err != nil {
				return c.Redirect(302, "/login")
			}
			token, err := base64.StdEncoding.DecodeString(cookie.Value)
			if err != nil {
				return c.Redirect(302, "/login")
			}
			username := string(token[:bytes.IndexByte(token, ':')])
			password := string(token[bytes.IndexByte(token, ':')+1:])
			if username != os.Getenv("API_USERNAME") || password != os.Getenv("API_PASSWORD") {
				return c.Redirect(302, "/login")
			}
			return next(c)
		}
	}
}
