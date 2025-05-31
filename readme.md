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

## API Documentation

### Projects

#### List Projects
```http
GET /api/v1/compose
```

Response body:
```json
["project1", "project2"]
```

### Services

#### List Services
```http
GET /api/v1/compose/{name}/service
```

Parameters:
- `name` (path, required) - Project name
- `service` (query, optional, repeatable) - Filter specific services

Response body:
```json
{
  "service1": {
    "Name": "service1",
    "State": "running",
    "Image": "nginx:latest",
    "Expected": 1,
    "Running": 1
  },
  "service2": {
    "Name": "service2",
    "State": "running",
    "Image": "redis:latest",
    "Expected": 1,
    "Running": 1
  }
}
```

Example Requests:
```bash
# Get all services including stopped ones
curl -X GET "http://mandok.local/api/v1/compose/myproject/service"

# Get specific services only
curl -X GET "http://mandok.local/api/v1/compose/myproject/service?service=web&service=db"
```

#### Get Service Details
```http
GET /api/v1/compose/{name}/service/{service}
```

Path Parameters:
- `name` (required) - Project name
- `service` (required) - Service name

#### Create/Update Service
```http
POST /api/v1/compose/{name}/service/{service}
```

Path Parameters:
- `name` (required) - Project name
- `service` (required) - Service name

Request body:
```json
{
  "image": "nginx:latest",
  "ports": ["80:80"],
  "environment": ["KEY=value"]
}
```

### Project Operations

#### Get Project Status
```http
GET /api/v1/compose/{name}/status
```

Path Parameters:
- `name` (required) - Project name

Query Parameters:
- `all` (optional) - boolean - Include stopped containers
- `service` (optional, repeatable) - string - Filter by service name(s)

Example Requests:
```bash
# Get all services including stopped ones
curl -X GET "http://mandok.local/api/v1/compose/myproject/status?all=true"

# Get specific services only
curl -X GET "http://mandok.local/api/v1/compose/myproject/status?service=web&service=db"
```

#### Start Project
```http
POST /api/v1/compose/{name}/start
```

Path Parameters:
- `name` (required) - Project name

Query Parameters:
- `restart` (optional) - boolean - Force restart containers
- `pull` (optional) - boolean - Pull images before starting
- `service` (optional, repeatable) - string - Start specific service(s)

#### Stop Project
```http
POST /api/v1/compose/{name}/stop
```

Path Parameters:
- `name` (required) - Project name

Query Parameters:
- `service` (optional, repeatable) - string - Stop specific service(s)

#### Delete Project
```http
DELETE /api/v1/compose/{name}
```

Path Parameters:
- `name` (required) - Project name

### Routing

#### Create Service Route
```http
POST /api/v1/compose/{name}/route/{service}
```

Path Parameters:
- `name` (required) - Project name
- `service` (required) - Service name

Request body:
```json
{
  "domain": "app.example.com",
  "path": "/api",
  "port": 8080,
  "ssl": true
}
```

#### Delete Service Route
```http
DELETE /api/v1/compose/{name}/route/{service}
```

Path Parameters:
- `name` (required) - Project name
- `service` (required) - Service name

#### List Routes
```http
GET /api/v1/compose/{name}/route
```

Path Parameters:
- `name` (required) - Project name

### Environment Variables

#### Set Environment Variables
```http
POST /api/v1/compose/{name}/envs
```

Path Parameters:
- `name` (required) - Project name

Request body:
```json
[
  {
    "key": "DATABASE_URL",
    "val": "postgresql://localhost:5432/db",
    "secret": false
  }
]
```

#### Get Environment Variables
```http
GET /api/v1/compose/{name}/envs
```

Path Parameters:
- `name` (required) - Project name

#### Delete Environment Variable
```http
DELETE /api/v1/compose/{name}/envs/{envname}
```

Path Parameters:
- `name` (required) - Project name
- `envname` (required) - Environment variable name

### Logs

#### Stream Service Logs
```http
GET /api/v1/compose/{name}/service/{service}/logs
```

Path Parameters:
- `name` (required) - Project name
- `service` (required) - Service name

Query Parameters:
- `tail` (optional) - integer - Number of historical log lines (default: 10)

