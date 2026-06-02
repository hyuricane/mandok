package compose

import (
	"context"
	"strings"
	"time"

	"github.com/docker/compose/v2/pkg/api"
)

type DockerComposeEvent struct {
	Action     string              `json:"action"`
	Service    string              `json:"service"`
	attributes `json:"attributes"` // embedded struct
	Time       string              `json:"time"`
	ID         string              `json:"id"`
	Type       string              `json:"type"`
}

type attributes struct {
	Name string `json:"name"`
}

func EventStreams(projectDir string) (chan DockerComposeEvent, func(), error) {
	ctx := context.Background()
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan DockerComposeEvent, 100)

	cancelableContext, cancel := context.WithCancel(ctx)

	go getAPI().Events(cancelableContext, project.Name, api.EventsOptions{
		Consumer: func(event api.Event) error {
			// ignore execs (presumed healthcheck), not an important docker event for us
			ignoredExecPrefixes := []string{"exec_create: ", "exec_start: ", "exec_die"} // these are common pattern for healthchecks
			for _, prefix := range ignoredExecPrefixes {
				if strings.HasPrefix(event.Status, prefix) {
					return nil
				}
			}

			dEvent := DockerComposeEvent{
				Action:  event.Status,
				Service: event.Service,
				attributes: attributes{
					Name: event.Attributes["name"],
				},
				Time: event.Timestamp.Format(time.RFC3339),
			}
			if event.Container != "" {
				dEvent.Type = "container"
			}

			ch <- dEvent
			return nil
		},
	})

	return ch, cancel, nil
}
