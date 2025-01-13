package compose

import (
	"context"
	"os/exec"
	"strconv"
)

func LogStream(projectDir, service string, tail int) (chan string, func(), error) {
	args := []string{"logs", "-f"}
	if tail > -1 {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	args = append(args, service)
	cmd := exec.CommandContext(context.Background(), "docker-compose", args...)
	cmd.Dir = projectDir
	return docExecStream(cmd)
}
