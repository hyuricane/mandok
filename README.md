# ManDok 

Simple API to restart docker compose container

## Requirements
### build
- golang
### running
- docker
- docker-compose


## environment variables
 - DOCKER_HOST=default to `unix:///var/run/docker.sock`
 - REGISTRY_AUTHS=username:password@registryhost,... your docker registry auths
 - WORKDIRS=projectname:/path/to/project;... mapping project name to project folder, where docker-compose.yml or compose.yml is located
 - API_USERNAME=basic auth username
 - API_PASSWORD=basic auth password
 - PORT=port to bind to default 8080
 - IP=ip address to bind to defaults to `0.0.0.0`, all ip addresses
 - DOCKER_COMPOSE_CMD_NAME=docker compose command name default to `docker-compose`, you can use `docker compose`

## running the application

### standalone
run `./dist`

### pm2
run `pm2 start ./dist/mandok --name=mandok --exec-mode=fork --autorestart`

### docker
you can deploy this with docker but you have to:
- mount `docker.sock` to this container, preverably to `/var/run/docker.sock`
- mount each docker-compose projects.
- map `WORKDIRS` of each projects.
- map `REGISTRY_AUTHS` of each docker images repository you use

this is the docker-compose.yml example:
```yaml
services:
  mandok:
    build: ./mandok
    environment:
      PORT: 80
      REGISTRY_AUTHS: "username:password@registryhost"
      WORKDIRS: "myproject:/usr/composes/myproject/;"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ../myproject/:/usr/composes/myproject/
    ports:
      - "8080:80"
```

### linux service
here is an example of linux service config file
```service
[Unit]
Description=Mandok service
After=docker.service

[Service]
WorkingDirectory=/path/to/your/service/directory
ExecStart=/path/to/your/service/executable
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target

```
`WorkingDirectory` is where your .env file is located, or 

here is what you must do:
- save that as `mandok.service` at `/etc/systemd/system/`
- run `sudo systemctl daemon-reload`
- run `sudo systemctl enable mandok`
- run `sudo systemctl start mandok`

## notes

only containers with `mandok=visible` labels is visible to this application

this is docker-compose.yml example of managed containers:
```yaml
services:
  myproject:
    image: my.image.repo/images:latest
    labels:
      - "mandok=visible"
```

