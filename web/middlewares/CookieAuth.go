package middlewares

import (
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
			claims, err := JwtValidateToken(cookie.Value)
			if err != nil {
				handleHTMXRedirect(c, loginPath)
				return c.Redirect(302, loginPath)
			}
			if claims.Username != conf.AppConfig.APIUsername {
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
