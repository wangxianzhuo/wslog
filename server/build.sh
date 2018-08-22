#!/bin/bash -x

DOCKER_IMGAE_NAME="wslog-server"
DOCKER_IMGAE_TAG="latest"

CGO_ENABLED=0 go build -a -installsuffix cgo -o build/wslog main.go

docker build -t ${DOCKER_IMGAE_NAME}:${DOCKER_IMGAE_TAG} .

docker save ${DOCKER_IMGAE_NAME}:${DOCKER_IMGAE_TAG} | gzip > build/${DOCKER_IMGAE_NAME}.tar.gz