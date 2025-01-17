package compose

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
)

type DockerComposeEvent struct {
	Action  string `json:"action"`
	Service string `json:"service"`
	Time    string `json:"time"`
	ID      string `json:"id"`
	Type    string `json:"type"`
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
	go func() {
		for line := range out {
			event := DockerComposeEvent{}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				log.Printf("[ERROR] %v", err)
				continue
			}
			ch <- event
		}
		close(ch)
	}()
	return ch, cancel, nil
}
