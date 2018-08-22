#!/bin/bash -x

DOCKER_IMGAE_NAME="wslog-ui"
DOCKER_IMGAE_TAG="latest"
docker build -t ${DOCKER_IMGAE_NAME}:${DOCKER_IMGAE_TAG} .

mkdir build

docker save ${DOCKER_IMGAE_NAME}:${DOCKER_IMGAE_TAG} | gzip > build/${DOCKER_IMGAE_NAME}.tar.gz
zip ${DOCKER_IMGAE_NAME}.zip -j build/${DOCKER_IMGAE_NAME}.tar.gz config.js
mv ${DOCKER_IMGAE_NAME}.zip build