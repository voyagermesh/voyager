#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

pushd $DOCKERFILE_DIR
cp $REPO_ROOT/dist/voyager/voyager-linux-amd64 voyager
chmod 755 voyager

DOCKERFILE="Dockerfile"

if [ "$IMAGE_TYPE" == "debug" ]; then
    cp Dockerfile Dockerfile.debug
    sed -i 's/USER nobody:nobody/USER root:root/' Dockerfile.debug
    echo "RUN apk --no-cache add curl" >> Dockerfile.debug
    DOCKERFILE="Dockerfile.debug"
fi

# download auth-request.lua
curl -fsSL -o auth-request.lua https://raw.githubusercontent.com/appscode/haproxy-auth-request/v1.8.12/auth-request.lua

cmd="docker build -t $DOCKER_REGISTRY/$IMAGE_NAME:$TAG-$IMAGE_TYPE -f $DOCKERFILE ."
echo $cmd; $cmd

rm voyager auth-request.lua

if [ "$IMAGE_TYPE" == "debug" ]; then
    rm Dockerfile.debug
fi

popd
