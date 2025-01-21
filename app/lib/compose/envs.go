package compose

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containerd/continuity/fs"
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
		log.Printf("[DEBUG] read %s error: %v", filepath.Join(projectDir, fileName), err)
		log.Printf("[DEBUG] isMasked %t", masked)
		if os.IsNotExist(err) {
			if masked { // just copy .env file
				log.Printf("[DEBUG] try read %s", filepath.Join(projectDir, ".env"))
				if _, err1 := os.Stat(filepath.Join(projectDir, ".env")); err1 != nil {
					log.Printf("[DEBUG] read %s error: %v", filepath.Join(projectDir, ".env"), err1)
					return plain, secret, err1
				}
				log.Printf("[DEBUG] can read %s", filepath.Join(projectDir, ".env"))
				err = fs.CopyFile(filepath.Join(projectDir, fileName), filepath.Join(projectDir, ".env"))
				if err != nil {
					return plain, secret, nil
				}
				return ReadEnvFile(projectDir, masked)
			}
			return plain, secret, nil
		}
		return nil, nil, err
	}
	lines := bytes.Split(payload, []byte("\n"))
	isPlain := true
	for _, bline := range lines {
		if len(bline) == 0 {
			continue
		}
		line := strings.TrimSpace(string(bline))
		if line == "### secret" {
			isPlain = false
			continue
		} else if line == "### compound-plain" {
			isPlain = true
			continue
		} else if line == "### compound-secret" {
			isPlain = false
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if isPlain {
			parts := strings.Split(line, "=")
			if len(parts) != 2 {
				continue
			}
			plain[string(parts[0])] = string(parts[1])
		} else {
			parts := strings.Split(line, "=")
			if len(parts) != 2 {
				continue
			}
			secret[parts[0]] = parts[1]
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
