#!/bin/bash -x

CGO_ENABLED=0 go build -a -installsuffix cgo -o build/wslog main.go &&
docker build -t wslog . &&
docker save wslog:latest | gzip > build/wslog.tar.gz