package middlewares

import (
	"bytes"
	"encoding/base64"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/conf"
)

func CookieAuth(loginPaths ...string) echo.MiddlewareFunc {
	loginPath := "/login"
	if len(loginPaths) > 0 {
		loginPath = loginPaths[0]
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("token")
			if err != nil {
				handleHTMXRedirect(c, loginPath)
				return c.Redirect(302, loginPath)
			}
			token, err := base64.StdEncoding.DecodeString(cookie.Value)
			if err != nil {
				handleHTMXRedirect(c, loginPath)
				return c.Redirect(302, loginPath)
			}
			username := string(token[:bytes.IndexByte(token, ':')])
			password := string(token[bytes.IndexByte(token, ':')+1:])
			if username != conf.AppConfig.APIUsername || password != conf.AppConfig.APIPassword {
				handleHTMXRedirect(c, loginPath)
				return c.Redirect(302, loginPath)
			}
			return next(c)
		}
	}
}

func handleHTMXRedirect(c echo.Context, loginPath string) {
	isHtmx := c.Request().Header.Get("HX-Request") == "true"
	if !isHtmx {
		return
	}
	c.Response().Header().Add("HX-Redirect", loginPath)
	c.Response().Header().Add("HX-Replace-URL", loginPath)
}
