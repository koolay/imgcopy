# Copy docker image to remote 

## Build and run

```bash

$ docker build -t imgcopy .
$ docker run --rm  -e DOCKER_CONFIG_FILE=/tmp/docker.config.json -e DOCKER_AUTH_NAME=hw -e DOCKER_AUTH_TOKEN=aa -p 8080:8080 imgcopy

```

- Open in browser: http://localhost:8080
- Request to copy

```bash
http://localhost:8080/api/copy?src=docker.io/library/busybox:latest&dest=my-registry.com/mirr/busybox:latest
```
