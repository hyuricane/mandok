package hx

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
)

type DockerComposeEvent struct {
	Action  string `json:"action"`
	Service string `json:"service"`
	Time    string `json:"time"`
	ID      string `json:"id"`
	Type    string `json:"type"`
}

func (e DockerComposeEvent) String() string {
	return e.Action + " " + e.Type + " " + e.Service
}
func getEvents(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/hx/")
	}
	ch, cancel, err := compose.EventStreams(projectDir)
	if err != nil {
		return err
	}
	defer cancel()

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case eventJson := <-ch:
			event := DockerComposeEvent{}
			err := json.Unmarshal([]byte(eventJson), &event)
			if err != nil {
				return err
			}
			c.Response().Write([]byte("event: docker-compose\n"))
			c.Response().Write([]byte("data: " + event.String() + "\n\n"))
			c.Response().Flush()
		}
	}
}
