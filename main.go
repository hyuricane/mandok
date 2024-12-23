package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"inovasiriset.co.id/docker/manager/app/lib/compose"
	"inovasiriset.co.id/docker/manager/web"
)

func main() {
	godotenv.Load()
	if os.Getenv("TRAEFIK") == "true" {
		ssl := os.Getenv("TRAEFIK_SSL") == "true"
		initTraefik(ssl)
	}
	log.Fatal(web.ListenHttp())
}

func initTraefik(ssl bool) error {
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

	// // init docker network traefik as external
	// err := exec.Command(
	// 	"docker", "network", "create", "traefik",
	// ).Run()
	// if err != nil {
	// 	return err
	// }

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
				"name": "traefik",
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
	err = compose.StartProject(projectDir, false, false)
	return err
}
