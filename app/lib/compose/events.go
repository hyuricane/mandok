package compose

import (
	"context"
	"os/exec"
)

func EventStreams(projectDir string) (chan string, func(), error) {
	args := []string{"events", "--json"}
	cmd := exec.CommandContext(context.Background(), "docker-compose", args...)
	cmd.Dir = projectDir
	return docExecStream(cmd)
}
