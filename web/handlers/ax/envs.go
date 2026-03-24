package ax

import (
	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/templates/axs/components"
)

func envs(c echo.Context) error {
	isAx := c.Request().Header.Get("X-Alpine-Request") == "true"
	if !isAx {
		return c.Redirect(302, "/ax")
	}

	projectName := c.Param("project")
	format := c.QueryParam("format")
	if format != "yaml" {
		format = "json"
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		c.Response().Header().Add("X-Alpine-Redirect", "/ax/")
		return c.Redirect(302, "/ax/")
	}
	project, err := compose.GetProject(projectDir, true)
	if err != nil {
		return components.Envs(projectName, nil, err).Render(c.Request().Context(), c.Response().Writer)
	}

	if format != "yaml" {
		format = "json"
	}

	if project == nil {
		return components.Envs(projectName, nil, nil).Render(c.Request().Context(), c.Response().Writer)
	}

	envVals, err := compose.ReadEnvFile(projectDir, true)

	return components.Envs(projectName, envVals, err).Render(c.Request().Context(), c.Response().Writer)
}

func setEnv(c echo.Context) error {
	isAx := c.Request().Header.Get("X-Alpine-Request") == "true"
	if !isAx {
		return c.Redirect(302, "/ax")
	}
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	name := c.FormValue("name")
	value := c.FormValue("value")
	envs := c.FormValue("envs")
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
	}
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return components.Envs(projectName, nil, err).Render(c.Request().Context(), c.Response().Writer)
	}
	secrets, err := compose.GetExistingSecretsEnvs(projectDir)
	if err != nil {
		return components.Envs(projectName, nil, err).Render(c.Request().Context(), c.Response().Writer)
	}

	if name != "" {
		envVals = append(envVals, compose.EnvVal{
			Key: name,
			Val: value,
		})
	}
	if len(envs) > 0 {
		newEnvVals, err := compose.ReadEnvsFromBytes([]byte(envs))
		if err != nil {
			return components.Envs(projectName, nil, err).Render(c.Request().Context(), c.Response().Writer)
		}
		envVals = append(envVals, newEnvVals...)
	}

	err = compose.WriteEnvFile(projectDir, envVals, secrets)
	maskedEnvVals, _ := compose.ReadEnvFile(projectDir, true)
	return components.Envs(projectName, maskedEnvVals, err).Render(c.Request().Context(), c.Response().Writer)
}

func setEnvSecret(c echo.Context) error {
	projectName := c.Param("project")
	name := c.Param("name")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
	}
	err := compose.SetEnvSecret(projectDir, name, true)
	maskedEnvVals, _ := compose.ReadEnvFile(projectDir, true)
	return components.Envs(projectName, maskedEnvVals, err).Render(c.Request().Context(), c.Response().Writer)
}

func deleteEnv(c echo.Context) error {
	projectName := c.Param("project")
	name := c.Param("name")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
	}
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return components.Envs(projectName, nil, err).Render(c.Request().Context(), c.Response().Writer)
	}
	exists := map[string]int{}
	for i, v := range envVals {
		exists[v.Key] = i
	}
	if i, ok := exists[name]; ok {
		envVals = append(envVals[:i], envVals[i+1:]...)
	}
	err = compose.WriteEnvFile(projectDir, envVals, nil)
	maskedEnvVals, _ := compose.ReadEnvFile(projectDir, true)
	return components.Envs(projectName, maskedEnvVals, err).Render(c.Request().Context(), c.Response().Writer)
}
