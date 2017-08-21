#!/usr/bin/env bash

VERSION=2.0

build() {
    rm -rf dist/*
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/server -a -ldflags '-linkmode external -extldflags -static -w' *.go
}

build_docker() {
    cp Dockerfile dist/
    docker build -t appscode/test-server:${VERSION} ./dist
}

docker_push() {
    docker push appscode/test-server:${VERSION}
}

all() {
    build
    build_docker
    docker_push
}

if [ $# -eq 0 ]; then
	all
	exit 0
fi

case "$1" in
    build)
      build
      ;;
	compile)
		go install ./...
		;;
	serve)
		go install ./...
		test-server
		;;
	docker)
	    build_docker
	    ;;
	push)
		build
		build_docker
		docker_push
		;;
	*)  echo $"Usage: $0 {compile|build|serve|push}"
		RETVAL=1
		;;
esac