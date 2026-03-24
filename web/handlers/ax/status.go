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

	statuses, err := compose.GetStatus(projectDir)
	return components.Statuses(projectName, statuses, err).Render(c.Request().Context(), c.Response().Writer)
}
