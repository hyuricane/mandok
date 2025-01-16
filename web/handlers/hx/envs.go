package hx

import (
	"path"
	"strings"

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
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	if envs != "" {
		newPlains := map[string]string{}
		lines := strings.Split(envs, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				continue
			}
			if line == "" {
				continue
			}
			linec := strings.SplitN(line, "=", 2)
			if len(linec) == 2 {
				newPlains[linec[0]] = linec[1]
			}
		}
		for k, v := range newPlains {
			plain[k] = v
			delete(secret, k)
		}
	}
	if name != "" {
		if path.Base(c.Path()) == "plain" {
			plain[name] = value
			delete(secret, name)
		} else {
			secret[name] = value
			delete(plain, name)
		}
	}

	err = compose.WriteEnvFile(projectDir, plain, secret)
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
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	if v, ok := plain[name]; ok {
		secret[name] = v
		delete(plain, name)
	}
	err = compose.WriteEnvFile(projectDir, plain, secret)
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
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	delete(plain, name)
	delete(secret, name)
	err = compose.WriteEnvFile(projectDir, plain, secret)
	if err != nil {
		return err
	}

	c.Response().Header().Add("HX-Trigger", "updateEnvs")
	return nil
}
