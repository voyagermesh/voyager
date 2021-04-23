#!/usr/bin/env bash

# Copyright AppsCode Inc. and Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DOCKER_REGISTRY=appscode
VERSION=2.4

build() {
    rm -rf dist/*
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dist/server *.go
}

build_docker() {
    cp Dockerfile dist/
    docker build --pull -t ${DOCKER_REGISTRY}/test-server:${VERSION} ./dist
}

docker_push() {
    docker push ${DOCKER_REGISTRY}/test-server:${VERSION}
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
    *)
        echo $"Usage: $0 {compile|build|serve|push}"
        RETVAL=1
        ;;
esac
