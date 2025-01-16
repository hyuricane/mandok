package compose

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dry run docker compose up
func TryProject(projectDir string, configFileName string) error {
	args := []string{"--dry-run"}
	if configFileName != "" {
		args = append(args, "-f", configFileName)
	}
	args = append(args, "up", "-d")
	cmd := exec.Command("docker-compose", args...)
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

func ReadEnvFile(projectDir string, masked bool) (plain map[string]string, secret map[string]string, err error) {
	fileName := ".env"
	if masked {
		fileName = "masked.env"
	}
	payload, err := os.ReadFile(filepath.Join(projectDir, fileName))
	plain = map[string]string{}
	secret = map[string]string{}
	if err != nil {
		if os.IsNotExist(err) {
			return plain, secret, nil
		}
		return nil, nil, err
	}
	lines := bytes.Split(payload, []byte("\n"))
	isPlain := true
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if string(line) == "### secret" {
			isPlain = false
			continue
		} else if string(line) == "### compound-plain" {
			isPlain = true
			continue
		} else if string(line) == "### compound-secret" {
			isPlain = false
			continue
		}
		if isPlain {
			parts := bytes.Split(line, []byte("="))
			if len(parts) != 2 {
				continue
			}
			plain[string(parts[0])] = string(parts[1])
		} else {
			parts := bytes.Split(line, []byte("="))
			if len(parts) != 2 {
				continue
			}
			secret[string(parts[0])] = string(parts[1])
		}
	}
	return plain, secret, nil
}

func WriteEnvFile(projectDir string, plain, secret map[string]string) error {
	realBs := bytes.Buffer{}
	maskedBs := bytes.Buffer{}
	compoundPlain := map[string]string{}
	compoundSecret := map[string]string{}
	for k, v := range plain {
		if strings.Contains(v, "${") {
			compoundPlain[k] = v
			delete(plain, k)
		}
	}
	for k, v := range secret {
		if strings.Contains(v, "${") {
			compoundSecret[k] = v
			delete(secret, k)
		}
	}
	for k, v := range plain {
		realBs.WriteString(k)
		realBs.WriteString("=")
		realBs.WriteString(v)
		realBs.WriteString("\n")

		maskedBs.WriteString(k)
		maskedBs.WriteString("=")
		maskedBs.WriteString(v)
		maskedBs.WriteString("\n")
	}
	realBs.WriteString("\n### secret\n")
	maskedBs.WriteString("\n### secret\n")
	for k, v := range secret {
		realBs.WriteString(k)
		realBs.WriteString("=")
		realBs.WriteString(v)
		realBs.WriteString("\n")

		maskedBs.WriteString(k)
		maskedBs.WriteString("=******\n")
	}

	realBs.WriteString("\n### compound-plain\n")
	maskedBs.WriteString("\n### compound-plain\n")
	for k, v := range compoundPlain {
		realBs.WriteString(k)
		realBs.WriteString("=")
		realBs.WriteString(v)
		realBs.WriteString("\n")

		maskedBs.WriteString(k)
		maskedBs.WriteString("=")
		maskedBs.WriteString(v)
		maskedBs.WriteString("\n")
	}
	realBs.WriteString("\n### compound-secret\n")
	maskedBs.WriteString("\n### compound-secret\n")
	for k, v := range compoundSecret {
		realBs.WriteString(k)
		realBs.WriteString("=")
		realBs.WriteString(v)
		realBs.WriteString("\n")

		maskedBs.WriteString(k)
		maskedBs.WriteString("=******\n")
	}
	err := os.WriteFile(filepath.Join(projectDir, ".env"), realBs.Bytes(), 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(projectDir, "masked.env"), maskedBs.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}
