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
	isHtmx := c.Request().Header.Get("Hx-Request") == "true"
	if !isHtmx {
		return c.Redirect(302, "/hx")
	}
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	name := c.FormValue("name")
	value := c.FormValue("value")
	envs := c.FormValue("envs")
	if projectDir == "" {
		return c.Redirect(302, "/hx/")
	}
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	secrets, err := compose.GetExistingSecretsEnvs(projectDir)
	if err != nil {
		return err
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
			return err
		}
		envVals = append(envVals, newEnvVals...)
	}

	err = compose.WriteEnvFile(projectDir, envVals, secrets)
	if err != nil {
		return err
	}
	c.Response().Header().Add("HX-Trigger", "updateEnvs")
	return nil
}

func setEnvSecret(c echo.Context) error {
	projectName := c.Param("project")
	name := c.Param("name")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/hx/")
	}
	err := compose.SetEnvSecret(projectDir, name, true)
	if err != nil {
		return err
	}
	c.Response().Header().Add("HX-Trigger", "updateEnvs")
	return nil
}

func deleteEnv(c echo.Context) error {
	projectName := c.Param("project")
	name := c.Param("name")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/hx/")
	}
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	exists := map[string]int{}
	for i, v := range envVals {
		exists[v.Key] = i
	}
	if i, ok := exists[name]; ok {
		envVals = append(envVals[:i], envVals[i+1:]...)
	}
	err = compose.WriteEnvFile(projectDir, envVals, nil)
	if err != nil {
		return err
	}

	c.Response().Header().Add("HX-Trigger", "updateEnvs")
	return nil
}
