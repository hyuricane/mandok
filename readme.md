# Mandok - Docker Compose Management Tool

A Docker container for managing multiple Docker Compose projects with optional Traefik integration for routing and SSL management.

## Features

- Manage multiple Docker Compose projects
- Optional Traefik integration for:
  - Automatic routing
  - SSL/TLS support (self-signed or Let's Encrypt)
  - HTTP to HTTPS redirection
- Multiple registry authentication support
- API authentication
- Configurable through environment variables

## Environment Variables

- `TZ`: time zone, default: *Asia/Jakarta*
- `API_USERNAME`: api username for Mandok authentication
- `API_PASSWORD`: api password for Mandok authentication
- `REGISTRY_AUTHS`: list of docker repositories auth, format: *username:password@registryhost;username2:password2@registryhost2*
- `WORKDIRS`: list of paths to docker compose projects
- `TRAEFIK`: enable Traefik integration, boolean, default: *false*
- `TRAEFIK_SSL`: enable SSL/TLS support, boolean, default: *false*
- `TRAEFIK_HTTP_ENTRYPOINT`: traefik HTTP entrypoint name, default: *web*
- `TRAEFIK_HTTPS_ENTRYPOINT`: traefik HTTPS entrypoint name, default: *websecure*
- `NETWORK`: traefik docker network name
- `MANDOK_DOMAIN`: mandok domain, eg: *mandok.local*
- `LETSENCRYPT_EMAIL`: email for Let's Encrypt certificates
- `HTTP_PORT`: host HTTP port mapping, default: *80*
- `HTTPS_PORT`: host HTTPS port mapping, default: *443*

## Deployment Options

### Basic Setup (Without Traefik)

1. Create a minimal docker-compose.yml:
```yaml
services:
  mandok:
    image: registry.gitlab.com/inovasirisetindonesia/modules/manage-docker:dev-compose
    environment:
      TZ: ${TZ:-Asia/Jakarta}
      REGISTRY_AUTHS: ${REGISTRY_AUTHS}
      WORKDIRS: ${WORKDIRS}
      API_USERNAME: ${API_USERNAME}
      API_PASSWORD: ${API_PASSWORD}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./projects:/usr/src/app/projects
```

### With Traefik Integration

1. Follow the full docker-compose.yml example in the How To section
2. For self-signed certificates:

```sh
# Create certificates directory
mkdir -p ./traefik/certs
cd ./traefik/certs

# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout cert.key -out cert.crt

# Create required files
touch ../acme.json
chmod 600 ../acme.json

# Create Traefik TLS configuration
mkdir -p ../configs
```

3. Create `./traefik/configs/tls.yml`:
```yaml
tls:
  stores:
    default:
      defaultCertificate:
        certFile: /certs/cert.crt
        keyFile: /certs/cert.key
```

4. For Let's Encrypt support:
   - Uncomment the Let's Encrypt sections in docker-compose.yml
   - Set `TRAEFIK_SSL=true`
   - Configure `LETSENCRYPT_EMAIL`

## Usage

1. Set required environment variables
2. Start the services:
```sh
docker-compose up -d
```

3. Access the Mandok API at:
   - HTTP: http://${MANDOK_DOMAIN}
   - HTTPS (if enabled): https://${MANDOK_DOMAIN}

## Notes

- When using Traefik, ensure the specified network exists
- API authentication is required for security
- Multiple Docker registries can be configured through `REGISTRY_AUTHS`
- Projects are mounted in `/usr/src/app/projects`

