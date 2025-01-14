package hx

import (
	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/templates/htmxs/components"
)

func status(c echo.Context) error {
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
		return components.Statuses(projectName, map[string]compose.ServiceStatus{}, err).Render(c.Request().Context(), c.Response().Writer)
	}

	if format != "yaml" {
		format = "json"
	}

	if project == nil {
		return components.Statuses(projectName, map[string]compose.ServiceStatus{}, err).Render(c.Request().Context(), c.Response().Writer)
	}

	statuses, err := compose.GetStatus(projectDir, true)
	if err != nil {
		return components.Statuses(projectName, map[string]compose.ServiceStatus{}, err).Render(c.Request().Context(), c.Response().Writer)
	}

	return components.Statuses(projectName, statuses, nil).Render(c.Request().Context(), c.Response().Writer)
}
