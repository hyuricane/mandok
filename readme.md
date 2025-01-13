
environments
---
- `TZ`: time zone, default: *Asia/Jakarta*
- `API_USERNAME`: api username
- `API_PASSWORD`: api password
- `REGISTRY_AUTHS`: list of docker reposiories auth, eg: *username:password@registryhost;username2:password2@registryhost2*
- `WORKDIRS`: list path to docker compose projects
- `TRAEFIK`: boolean, default: *false*
- `TRAEFIK_SSL`: boolean, default: *false*
- `TRAEFIK_HTTP_ENTRYPOINT`: traefik default http entrypoint, default: *web*
- `TRAEFIK_HTTPS_ENTRYPOINT`: traefik default https entrypoint, default: *websecure*
- `NETWORK`: traefik docker network
- `MANDOK_DOMAIN`: mandok domain, eg: *mandok.local*
- `LETSENCRYPT_EMAIL`: email for letsencrypt certificates
- `HTTP_PORT`: http port, default: *80*
- `HTTPS_PORT`: https port, default: *443*

how to: 
--- 
sample **docker-compose.yml**
```yaml
services:
  mandok:
    build: ./mandok
    environment:
      TZ: ${TZ:-Asia/Jakarta}
      REGISTRY_AUTHS: ${REGISTRY_AUTHS}
      WORKDIRS: ${WORKDIRS}

      API_USERNAME: ${API_USERNAME}
      API_PASSWORD: ${API_PASSWORD}

      TRAEFIK: ${TRAEFIK:-false}
      TRAEFIK_SSL: ${TRAEFIK_SSL:-false}
      TRAEFIK_HTTP_ENTRYPOINT: ${TRAEFIK_HTTP_ENTRYPOINT}
      TRAEFIK_HTTPS_ENTRYPOINT: ${TRAEFIK_HTTPS_ENTRYPOINT}
      NETWORK: ${NETWORK}
    labels:
      mandok: visible
      traefik.enable: true
      traefik.docker.network: ${NETWORK}
      traefik.http.services.mandok.loadbalancer.server.port: 80
      # http
      traefik.http.routers.mandok.rule: Host(`${MANDOK_DOMAIN}`)
      traefik.http.routers.mandok.entrypoints: ${TRAEFIK_HTTP_ENTRYPOINT:-web}
      # https
      traefik.http.routers.mandok-secure.rule: Host(`${MANDOK_DOMAIN}`)
      traefik.http.routers.mandok-secure.tls: true
      traefik.http.routers.mandok-secure.entrypoints: ${TRAEFIK_HTTPS_ENTRYPOINT:-websecure}
      # # letsencrypt
      # traefik.http.routers.mandok-secure.tls.certresolver: letsencrypt
      # redirect to https
      traefik.http.routers.mandok.middlewares: https_redirect
      traefik.http.middlewares.https_redirect.redirectscheme.scheme: https
    volumes:
      - ${DOCKER_SOCK:-/var/run/docker.sock}:/var/run/docker.sock
      - ./projects:/usr/src/app/projects
  traefik:
    command:
      - --log.level=DEBUG
      - --api.insecure=true
      - --providers.docker.exposedbydefault=false
      - --providers.docker=true
      - --entryPoints.${TRAEFIK_HTTP_ENTRYPOINT:-web}.address=:80
      - --entryPoints.${TRAEFIK_HTTPS_ENTRYPOINT:-websecure}.address=:443
      # # letsencrypt cert resolver
      # - --certificatesresolvers.letsencrypt.acme.tlschallenge=true
      # - --certificatesresolvers.letsencrypt.acme.email=${LETSENCRYPT_EMAIL}
      # - --certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json
      - --providers.file.directory=/configs
    image: traefik
    ports:
      - ${HTTP_PORT:-:80}:80
      - ${HTTPS_PORT:-:443}:443
    restart: always
    volumes:
      - "${DOCKER_SOCK:-/var/run/docker.sock}:/var/run/docker.sock"
      - "./traefik/acme.json:/letsencrypt/acme.json"
      - "./traefik/configs:/configs"
      # backup certificates
      - "./traefik/certs:/certs"
networks:
  default:
    name: ${NETWORK}
```


notes: deploying with backup self signed certificates
---

create certificates in `./traefik/certs`

```sh
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout cert.key -out cert.crt
```
create `./traefik/acme.json`

create `./traefik/configs/tls.yml`
```yaml
tls:
  stores:
    default:
      defaultCertificate:
        certFile: /certs/cert.crt
        keyFile: /certs/cert.key
```

