package v1

import (
	"log"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
)

const PROJECT_DIRS = "projects"

func RouteCompose(group *echo.Group) {
	group.POST("/:name", createProject)
	group.GET("/:name", getProject)
	group.GET("/:name/status", getStatus)
	group.POST("/:name/start", startProject)
	group.POST("/:name/stop", stopProject)
	group.DELETE("/:name", deleteProject)

	group.POST("/:name/route/:service", routeService)
	group.DELETE("/:name/route/:service", deleteRoute)

	group.POST("/:name/envs", setEnvs)
	group.GET("/:name/envs", getEnvs)
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

type ExpectedPSData struct {
	Service    string `json:"Service"`
	CreatedAt  string `json:"CreatedAt"`
	Image      string `json:"Image"`
	Status     string `json:"Status"`
	State      string `json:"State"`
	Size       string `json:"Size"`
	RunningFor string `json:"RunningFor"`
	ExitCode   int    `json:"ExitCode"`
}

func getStatus(c echo.Context) error {
	name := c.Param("name")
	projectDir := compose.HasProject(name)
	if projectDir == "" {
		return c.JSON(404, map[string]string{
			"message": "project not found",
		})
	}
	all := c.QueryParam("all")
	services := []string{}
	for k, v := range c.Request().URL.Query() {
		// keys = append(keys, k)
		if k == "service" {
			services = append(services, v...)
			continue
		}
	}
	status, err := compose.GetStatus(projectDir, all == "true", services...)
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

type RouteBodyService struct {
	Domain string                 `json:"domain"`
	Port   int                    `json:"port"`
	Sticky map[string]interface{} `json:"sticky"`
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
	body := RouteBodyService{}
	err := c.Bind(&body)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	err = compose.RouteService(projectDir, serviceName, body.Domain, body.Port, body.Sticky)
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

type EnvBody struct {
	Plain  map[string]string `json:"plain"`
	Secret map[string]string `json:"secret"`
}

func setEnvs(c echo.Context) error {
	name := c.Param("name")
	body := EnvBody{}
	err := c.Bind(&body)
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
	plain, secret, err := compose.ReadEnvFile(projectDir, false)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	// merge envs
	for k, v := range body.Plain {
		plain[k] = v
	}
	for k, v := range body.Secret {
		secret[k] = v
	}

	// write to env files
	err = compose.WriteEnvFile(projectDir, plain, secret)
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
	plain, secret, err := compose.ReadEnvFile(projectDir, true)
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]interface{}{
		"message": "ok",
		"plain":   plain,
		"secret":  secret,
	})
}
