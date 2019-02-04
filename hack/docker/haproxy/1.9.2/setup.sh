#!/bin/bash

set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

detect_tag $REPO_ROOT/dist/.tag

IMG=haproxy
TAG=1.9.2-$TAG
DOCKER_REGISTRY=${DOCKER_REGISTRY:-appscode}

build() {
  pushd $(dirname "${BASH_SOURCE}")
  cp $REPO_ROOT/dist/voyager/voyager-linux-amd64 voyager
  chmod +x voyager

  # download socklog (`socklog` not available for `stretch`, use `jessie` deb instead)
  curl -L -o socklog.deb http://ftp.us.debian.org/debian/pool/main/s/socklog/socklog_2.1.0-8_amd64.deb
  # download auth-request.lua
  curl -fsSL -o auth-request.lua https://raw.githubusercontent.com/appscode/haproxy-auth-request/v1.9.2/auth-request.lua

  local cmd="docker build --pull -t $DOCKER_REGISTRY/$IMG:$TAG ."
  echo $cmd
  $cmd
  rm voyager socklog.deb auth-request.lua
  popd
}

binary_repo $@
