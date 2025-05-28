package compose

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containerd/continuity/fs"
)

type EnvVal struct {
	Key    string `json:"key"`
	Val    string `json:"val"`
	Secret bool   `json:"secret"`
}

func (ev EnvVal) MarshalJSON() ([]byte, error) {
	type Alias EnvVal
	alias := &Alias{
		Key:    ev.Key,
		Val:    ev.Val,
		Secret: ev.Secret,
	}
	if ev.Secret {
		alias.Val = "*****"
	}

	return json.Marshal(alias)
}

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

func ReadEnvFile(projectDir string, masked bool) (vals []EnvVal, err error) {
	fileName := ".env"
	if masked {
		fileName = "masked.env"
	}
	payload, err := os.ReadFile(filepath.Join(projectDir, fileName))
	if os.IsNotExist((err)) {
		var f *os.File
		_, err = os.Create(filepath.Join(projectDir, fileName))
		if err != nil {
			return nil, err
		}
		f.Close()
		payload, err = os.ReadFile(filepath.Join(projectDir, fileName))
	}
	vals = []EnvVal{}
	if err != nil {
		log.Printf("[DEBUG] read %s error: %v", filepath.Join(projectDir, fileName), err)
		log.Printf("[DEBUG] isMasked %t", masked)
		if os.IsNotExist(err) {
			if masked { // just copy .env file
				log.Printf("[DEBUG] try read %s", filepath.Join(projectDir, ".env"))
				if _, err1 := os.Stat(filepath.Join(projectDir, ".env")); err1 != nil {
					log.Printf("[DEBUG] read %s error: %v", filepath.Join(projectDir, ".env"), err1)
					return vals, err1
				}
				log.Printf("[DEBUG] can read %s", filepath.Join(projectDir, ".env"))
				err = fs.CopyFile(filepath.Join(projectDir, fileName), filepath.Join(projectDir, ".env"))
				if err != nil {
					return vals, nil
				}
				return ReadEnvFile(projectDir, masked)
			}
			return vals, nil
		}
		return nil, err
	}
	vals, err = ReadEnvsFromBytes(payload)

	return
}

func ReadEnvsFromBytes(payload []byte) ([]EnvVal, error) {
	vals := []EnvVal{}
	lines := bytes.Split(payload, []byte("\n"))
	/**
	the format is
	KEYPLAIN_KEY1=PLAIN_VAL1
	### secret
	SECRET_KEY=SECRET_VAL
	PLAIN_KEY2=PLAIN_VAL2
	**/
	isSecret := false
	for _, bline := range lines {
		if len(bline) == 0 {
			continue
		}
		line := strings.TrimSpace(string(bline))
		if line == "### secret" {
			isSecret = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			isSecret = false
			continue
		}
		parts := strings.Split(line, "=")
		vals = append(vals, EnvVal{
			Key:    string(parts[0]),
			Val:    string(parts[1]),
			Secret: isSecret,
		})
		isSecret = false
	}
	return vals, nil
}

func WriteEnvFile(projectDir string, vals []EnvVal) error {
	realBs := bytes.Buffer{}
	maskedBs := bytes.Buffer{}
	for _, v := range vals {
		if v.Secret {
			realBs.WriteString("### secret\n")
			maskedBs.WriteString("### secret\n")
		}
		realBs.WriteString(v.Key)
		realBs.WriteString("=")
		realBs.WriteString(v.Val)
		realBs.WriteString("\n")

		maskedBs.WriteString(v.Key)
		if v.Secret {
			maskedBs.WriteString("=******\n")
		} else {
			maskedBs.WriteString("=")
			maskedBs.WriteString(v.Val)
			maskedBs.WriteString("\n")
		}
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
