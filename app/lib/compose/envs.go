package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type EnvVal struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func (ev EnvVal) MarshalJSON() ([]byte, error) {
	type Alias EnvVal
	alias := &Alias{
		Key: ev.Key,
		Val: ev.Val,
	}

	return json.Marshal(alias)
}

// dry run docker compose up
func TryProject(projectDir string, configFileName string, services ...string) error {
	ctx := context.Background()
	_, err := LoadProject(ctx, projectDir, LoadProjectOptions{
		ConfigFiles: []string{configFileName},
	})
	return err
}

func ReadEnvFile(projectDir string, masked bool) (vals []EnvVal, err error) {
	fileNames := []string{filepath.Join(projectDir, ".env")}
	if masked {
		fileNames = append(fileNames, filepath.Join(projectDir, "masked.env"))
	}
	for _, fileName := range fileNames {
		if _, err := os.Stat(fileName); err != nil && !os.IsNotExist(err) {
			os.WriteFile(fileName, []byte{}, 0644)
		}
	}
	envvals, err := godotenv.Read(fileNames...)
	if err != nil {
		return nil, err
	}
	for k, v := range envvals {
		vals = append(vals, EnvVal{
			Key: k,
			Val: v,
		})
	}
	return vals, nil
}

func GetExistingSecretsEnvs(projectDir string) (map[string]bool, error) {
	secretEnvs, err := godotenv.Read(filepath.Join(projectDir, "masked.env"))
	if err != nil {
		return nil, err
	}
	secrets := make(map[string]bool)
	for k := range secretEnvs {
		secrets[k] = true
	}
	return secrets, nil
}

func SetEnvSecret(projectDir string, key string, secret bool) error {
	envs, err := ReadEnvFile(projectDir, false)
	if err != nil {
		return err
	}
	secrets, err := GetExistingSecretsEnvs(projectDir)
	if err != nil {
		return err
	}
	secrets[key] = secret
	return WriteEnvFile(projectDir, envs, secrets)
}

func ReadEnvsFromBytes(payload []byte) ([]EnvVal, error) {
	vals := []EnvVal{}
	envvals, err := godotenv.Parse(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for k, v := range envvals {
		vals = append(vals, EnvVal{
			Key: k,
			Val: v,
		})
	}
	return vals, nil
}

func WriteEnvFile(projectDir string, vals []EnvVal, secrets map[string]bool) error {
	envmapPlain := make(map[string]string)
	envmapMasked := make(map[string]string)
	for _, v := range vals {
		envmapPlain[v.Key] = v.Val
		if secrets[v.Key] {
			envmapMasked[v.Key] = "******"
		}
	}
	payloadPlain, err := godotenv.Marshal(envmapPlain)
	if err != nil {
		return err
	}
	payloadMasked, err := godotenv.Marshal(envmapMasked)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(projectDir, ".env"), []byte(payloadPlain), 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(projectDir, "masked.env"), []byte(payloadMasked), 0644)
	if err != nil {
		return err
	}
	return nil
}
