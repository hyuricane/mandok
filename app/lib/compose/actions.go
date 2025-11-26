package compose

import (
	"bytes"
	"os/exec"
)

func StartProject(projectDir string, restart bool, pull bool, services ...string) error {
	args := []string{"up", "-d"}
	if restart {
		args = append(args, "--force-recreate")
	}
	if pull {
		args = append(args, "--pull", "always", "--build")
	}
	args = append(args, services...)
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
	cmd.Dir = projectDir
	_, err := doExec(cmd)
	if err != nil {
		return err
	}

	return nil
}

func StopProject(projectDir string, service ...string) error {
	args := []string{"stop"}
	args = append(args, service...)
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
	cmd.Dir = projectDir

	_, err := doExec(cmd)

	if err != nil {
		if cErr := NewComposeError(bytes.NewBufferString(err.Error())); cErr != nil {
			return cErr
		}

		return err
	}
	return nil
}

func DownProject(projectDir string) error {
	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = projectDir
	_, err := doExec(cmd)

	if err != nil {
		if cErr := NewComposeError(bytes.NewBufferString(err.Error())); cErr != nil {
			return cErr
		}
		return err
	}
	return nil
}
