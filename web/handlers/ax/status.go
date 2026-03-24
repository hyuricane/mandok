package ax

import (
	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/templates/axs/components"
)

func status(c echo.Context) error {
	isAx := c.Request().Header.Get("X-Alpine-Request") == "true"
	if !isAx {
		return c.Redirect(302, "/ax")
	}

	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		c.Response().Header().Add("X-Alpine-Redirect", "/ax/")
		return c.Redirect(302, "/ax/")
	}
	project, err := compose.GetProject(projectDir, true)
	if err != nil {
		return components.Statuses(projectName, map[string]compose.ServiceStatus{}, err).Render(c.Request().Context(), c.Response().Writer)
	}

	if project == nil {
		return components.Statuses(projectName, map[string]compose.ServiceStatus{}, err).Render(c.Request().Context(), c.Response().Writer)
	}

	statuses, err := compose.GetStatus(projectDir, true)
	return components.Statuses(projectName, statuses, err).Render(c.Request().Context(), c.Response().Writer)
}
