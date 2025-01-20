package compose

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"path"
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
	args := []string{"events", "--json"}
	cmd := exec.CommandContext(context.Background(), "docker-compose", args...)
	cmd.Dir = projectDir
	out, cancel, err := docExecStream(cmd)
	if err != nil {
		return nil, nil, err
	}
	ch := make(chan DockerComposeEvent, 100)
	projectName := path.Base(projectDir)
	go func() {
		for line := range out {
			event := DockerComposeEvent{}
			log.Printf("[DEBUG] event line %s", line)
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				log.Printf("[ERROR] %v", err)
				continue
			}
			// remove project name ({projectName}-{serviceName}-{instance}) from event.Name
			if len(event.attributes.Name) > len(projectName) {
				if event.attributes.Name[len(projectName)] == '-' {
					event.attributes.Name = event.attributes.Name[len(projectName)+1:]
				}
			}
			ch <- event
		}
		close(ch)
	}()
	return ch, cancel, nil
}
