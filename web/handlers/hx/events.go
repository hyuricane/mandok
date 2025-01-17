package hx

import (
	"time"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
)

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
	c.Response().Header().Set("Transfer-Encoding", "chunked")
	c.Response().WriteHeader(200)
	keepAlive := time.NewTicker(30 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-keepAlive.C:
			if _, err = c.Response().Writer.Write([]byte(":keep-alive\n\n")); err != nil {
				return err
			}
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if _, err = c.Response().Writer.Write([]byte("event: " + event.Type + "\n")); err != nil {
				return err
			}
			if _, err = c.Response().Writer.Write([]byte("data: <p remove-me=\"1s\" class=\"event\">" + event.Action + " " + event.Service + "</p>\n\n")); err != nil {
				return err
			}
		}
		c.Response().Flush()
	}
}
