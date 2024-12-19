package v1

import (
	"os"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func RouteRepo(group *echo.Group) {
	group.POST("/login", login)
}

func login(c echo.Context) error {
	// login to repo
	auth := RepoAuth{}
	err := c.Bind(&auth)
	if err != nil {
		return c.JSON(400, map[string]string{
			"message": err.Error(),
		})
	}
	commands := []string{"login"}
	if auth.Username != "" {
		commands = append(commands, "-u", auth.Username)
	}
	if auth.Password != "" {
		commands = append(commands, "-p", auth.Password)
	}
	if auth.Registry != "" {
		commands = append(commands, auth.Registry)
	}
	cmd := exec.Command("docker", commands...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return c.JSON(500, map[string]string{
			"message": err.Error(),
		})
	}
	return c.JSON(200, map[string]string{
		"message": "ok",
	})
}
