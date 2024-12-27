package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web"
)

func main() {
	godotenv.Load()
	if os.Getenv("TRAEFIK") == "true" {
		ssl := os.Getenv("TRAEFIK_SSL") == "true"
		err := initTraefik(ssl)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Fatal(web.ListenHttp())
}

func initTraefik(ssl bool) error {
	networkname := os.Getenv("NETWORK")
	if networkname == "" {
		networkname = "mandok"
	}

	sslCommands := []string{}
	if ssl {
		acmeEmail := os.Getenv("ACME_EMAIL")
		if acmeEmail == "" {
			return fmt.Errorf("ACME_EMAIL is not set")
		}
		sslCommands = append(sslCommands,
			"--certificatesresolvers.default.acme.httpchallenge.entrypoint=http",
			fmt.Sprintf("--certificatesresolvers.default.acme.email=\"%s\"", acmeEmail),
			"--certificatesResolvers.default.acme.storage=/letsencrypt/acme.json",
		)
	}

	projectConfig := compose.ComposeProjectYaml{
		Services: map[string]interface{}{
			"traefik": map[string]interface{}{
				"image": "traefik",
				"command": append([]string{
					"--log.level=DEBUG",
					"--api.insecure=true",
					"--providers.docker.exposedbydefault=false",
					"--providers.docker=true",
				}, sslCommands...),
				"ports": []string{
					"80:80",
					"443:443",
				},
				"volumes": []string{
					"/var/run/docker.sock:/var/run/docker.sock",
					"./acme.json:/letsencrypt/acme.json",
				},
				"restart": "always",
			},
		},
		Networks: map[string]interface{}{
			"default": map[string]interface{}{
				"name": networkname,
			},
		},
	}

	projectDir, err := compose.CreateProject("traefik", projectConfig)
	if err != nil {
		return err
	}
	acmeFilepath := filepath.Join(projectDir, "acme.json")
	// create acme file if not exists
	if _, err := os.Stat(acmeFilepath); os.IsNotExist(err) {
		f, err := os.Create(acmeFilepath)
		if err != nil {
			return err
		}
		f.Close()
	}
	log.Printf("[DEBUG] start traefik")
	err = compose.StartProject(projectDir, false, false)
	if err != nil {
		return err
	}

	cid, err := getCID()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] container id %s", cid)
	err = compose.AttachToDockerNetwork(networkname, cid)
	if err != nil {
		return err
	}

	// restart traefik for
	err = compose.StartProject(projectDir, true, false, "traefik")
	if err != nil {
		return err
	}
	return err
}

func getCID() (string, error) {
	cidExec := exec.Command("cat", "/proc/1/cpuset")
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cidExec.Stdout = buff
	cidExec.Stderr = buffErr
	err := cidExec.Run()
	if err != nil {
		if buffErr.Len() > 0 {
			err = fmt.Errorf("%s", buffErr.String())
		}
		return "", err
	}
	cid := path.Base(strings.TrimSpace(buff.String()))
	return cid, nil
}
