package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
)

const PROJECT_DIRS = "projects"

func RouteCompose(group *echo.Group) {
	group.GET("", getProjects)
	group.POST("/:name", createProject)
	group.GET("/:name", getProject)
	group.GET("/:name/service", getStatus)
	group.GET("/:name/service/:service", getService)
	group.POST("/:name/service/:service", createService)
	group.POST("/:name/start", startProject)
	group.POST("/:name/stop", stopProject)
	group.DELETE("/:name", deleteProject)

	group.POST("/:name/route/:service", routeService)
	group.DELETE("/:name/route/:service", deleteRoute)
	group.GET("/:name/route", getRoutes)

	group.POST("/:name/envs", setEnvs)
	group.GET("/:name/envs", getEnvs)
	group.DELETE("/:name/envs/:envname", deleteEnv)

	group.GET("/:name/service/:service/logs", getLogs)
}

func getProjects(c echo.Context) error {
	projects, err := compose.GetProjects()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, projects)
}

func createProject(c echo.Context) error {
	// read yaml file
	log.Printf("[DEBUG] create project %s", c.Param("name"))
	name := c.Param("name")
	body := compose.ComposeProjectYaml{}
	err := c.Bind(&body)
	if err != nil {
		return c.JSON(400, map[string]string{
			"message": err.Error(),
		})
	}
	_, err = compose.CreateProject(name, body)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func getProject(c echo.Context) error {
	name := c.Param("name")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	project, err := compose.GetProject(projectDir)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, project)
}

func createService(c echo.Context) error {
	projectName := c.Param("name")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}

	service := map[string]interface{}{}
	defer c.Request().Body.Close()
	err := json.NewDecoder(c.Request().Body).Decode(&service)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	err = compose.CreateService(projectDir, serviceName, service)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func getService(c echo.Context) error {
	projectName := c.Param("name")
	serviceName := c.Param("service")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	service, err := compose.GetService(projectDir, serviceName)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, service)
}

func getStatus(c echo.Context) error {
	name := c.Param("name")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	services := []string{}
	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			services = append(services, v...)
			continue
		}
	}
	status, err := compose.GetStatus(projectDir, true, services...)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]interface{}{
		"services": status,
	})
}

func startProject(c echo.Context) error {
	name := c.Param("name")
	restart := c.QueryParam("restart")
	pull := c.QueryParam("pull")
	services := []string{}
	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			services = append(services, v...)
			continue
		}
	}
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}

	err := compose.StartProject(projectDir, restart == "true", pull == "true", services...)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func stopProject(c echo.Context) error {
	name := c.Param("name")
	services := []string{}
	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			services = append(services, v...)
			continue
		}
	}
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}

	err := compose.StopProject(projectDir, services...)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func deleteProject(c echo.Context) error {
	name := c.Param("name")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	err := compose.DownProject(projectDir)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func routeService(c echo.Context) error {
	projectName := c.Param("name")
	serviceName := c.Param("service")
	if projectName == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	if serviceName == "" {
		return c.JSON(404, map[string]string{
			"message": "service not found",
		})
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	body := compose.ServiceRoute{}
	err := c.Bind(&body)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	err = compose.RouteService(projectDir, serviceName, body)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(200, map[string]interface{}{
		"message": "ok",
	})
}

func deleteRoute(c echo.Context) error {
	projectName := c.Param("name")
	serviceName := c.Param("service")
	if projectName == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	if serviceName == "" {
		return c.JSON(404, map[string]string{
			"message": "service not found",
		})
	}
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	err := compose.DeleteRoute(projectDir, serviceName)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func getRoutes(c echo.Context) error {
	name := c.Param("name")
	services := []string{}
	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			services = append(services, v...)
			continue
		}
	}
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	routes, err := compose.GetRoutes(projectDir, services...)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, routes)
}

func setEnvs(c echo.Context) error {
	name := c.Param("name")
	newEnvVals := []compose.EnvVal{}
	err := c.Bind(&newEnvVals)
	if err != nil {
		return c.JSON(400, map[string]string{
			"message": err.Error(),
		})
	}
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	// read from existing .env file
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	exists := map[string]int{}
	for i, v := range envVals {
		exists[v.Key] = i
	}
	// merge envs
	for _, v := range newEnvVals {
		if i, ok := exists[v.Key]; ok {
			envVals[i].Val = v.Val
			envVals[i].Secret = v.Secret
		} else {
			envVals = append(envVals, v)
		}
	}

	// write to env files
	err = compose.WriteEnvFile(projectDir, envVals)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func getEnvs(c echo.Context) error {
	name := c.Param("name")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	// read .env file
	envVals, err := compose.ReadEnvFile(projectDir, true)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]interface{}{
		"message": "ok",
		"envs":    envVals,
	})
}

func deleteEnv(c echo.Context) error {
	name := c.Param("name")
	envname := c.Param("envname")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	envVals, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	exists := map[string]int{}
	for i, v := range envVals {
		exists[v.Key] = i
	}
	if i, ok := exists[envname]; ok {
		envVals = append(envVals[:i], envVals[i+1:]...)
	}
	err = compose.WriteEnvFile(projectDir, envVals)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}

func getLogs(c echo.Context) error {
	name := c.Param("name")
	serviceName := c.Param("service")
	tail := 10
	if tails := c.QueryParam("tail"); tails != "" {
		if tailI, err := strconv.Atoi(tails); err == nil {
			tail = tailI
		}
	}
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	statuses, err := compose.GetStatus(projectDir, false, serviceName)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	status, ok := statuses[serviceName]
	if !ok {
		return c.JSON(404, map[string]string{
			"message": "service not found",
		})
	}
	if status.Running == 0 {
		return c.JSON(404, map[string]string{
			"message": "service not running",
		})
	}
	// get logs

	// return sse
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(200)

	ch, cancel, err := compose.LogStreamWithContext(c.Request().Context(), projectDir, serviceName, tail)

	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	defer cancel()
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case lines := <-ch:
			if lines == "" { // assume closed
				return nil
			}
			for _, line := range strings.Split(lines, "\n") {
				fmt.Fprintf(c.Response().Writer, "event: log\ndata: %s\n\n", line)
			}
			c.Response().Flush()
		}
	}
}
