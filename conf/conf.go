package conf

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ProjectDirs             string
	Network                 string
	Traefik                 bool
	TraefikSSL              bool
	TraefikHTTPEntrypoint   string
	TraefikHTTPSEntrypoint  string
	RegistryAuths           string
	APIUsername             string
	APIPassword             string
	Port                    string
	IP                      string
	Workdirs                string
	DockerComposeCmdName    string
}

var AppConfig *Config

func init() {
	if err := godotenv.Load(); err == nil {
		log.Printf("[INFO] .env file loaded")
	}

	AppConfig = &Config{
		ProjectDirs:             getEnv("PROJECT_DIRS", "projects"),
		Network:                 getEnv("NETWORK", "mandok"),
		Traefik:                 getEnvBool("TRAEFIK", false),
		TraefikSSL:              getEnvBool("TRAEFIK_SSL", false),
		TraefikHTTPEntrypoint:   getEnv("TRAEFIK_HTTP_ENTRYPOINT", "web"),
		TraefikHTTPSEntrypoint:  getEnv("TRAEFIK_HTTPS_ENTRYPOINT", "websecure"),
		RegistryAuths:           getEnv("REGISTRY_AUTHS", ""),
		APIUsername:             getEnv("API_USERNAME", ""),
		APIPassword:             getEnv("API_PASSWORD", ""),
		Port:                    getEnv("PORT", "8080"),
		IP:                      getEnv("IP", "0.0.0.0"),
		Workdirs:                getEnv("WORKDIRS", ""),
		DockerComposeCmdName:    getEnv("DOCKER_COMPOSE_CMD_NAME", "docker compose"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return fallback
}
