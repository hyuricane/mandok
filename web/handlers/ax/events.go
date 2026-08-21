package ax

import (
	"time"

	"github.com/labstack/echo/v4"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web/templates/axs/components"
)

func getEvents(c echo.Context) error {
	projectName := c.Param("project")
	projectDir := compose.HasProject(projectName)
	if projectDir == "" {
		return c.Redirect(302, "/ax/")
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
	c.Response().Header().Set("X-Accel-Buffering", "no")
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
			if _, err = c.Response().Writer.Write([]byte("event: " + event.Type + "\ndata: ")); err != nil {
				return err
			}
			if err = components.Event(event).Render(c.Request().Context(), c.Response().Writer); err != nil {
				return err
			}
			if _, err = c.Response().Writer.Write([]byte("\n\n")); err != nil {
				return err
			}
			if event.Type == "container" && (event.Action == "start" || event.Action == "die") {
				if _, err = c.Response().Writer.Write([]byte("event: updateStatus\n\n")); err != nil {
					return err
				}
			}
		}
		c.Response().Flush()
	}
}
