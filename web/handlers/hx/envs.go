package hx

import (
	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/templates/htmxs/components"
)

func envs(c echo.Context) error {
	isHtmx := c.Request().Header.Get("Hx-Request") == "true"
	if !isHtmx {
		return c.Redirect(302, "/hx")
	}

	projectName := c.Param("project")
	format := c.QueryParam("format")
	if format != "yaml" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		c.Response().Header().Add("HX-Redirect", "/hx/")
		c.Response().Header().Add("HX-Replace-URL", "/hx/")
		return c.Redirect(302, "/hx/")
	}
	project, err := compose.GetProject(projectDir, true)
	if err != nil {
		return components.Envs(projectName, nil, nil, err).Render(c.Request().Context(), c.Response().Writer)
	}

	if format != "yaml" {
		format = "json"
	}

	if project == nil {
		return components.Envs(projectName, nil, nil, nil).Render(c.Request().Context(), c.Response().Writer)
	}

	plain, secret, err := compose.ReadEnvFile(projectDir, true)

	return components.Envs(projectName, plain, secret, err).Render(c.Request().Context(), c.Response().Writer)
}
