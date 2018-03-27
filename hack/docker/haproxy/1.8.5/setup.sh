#!/bin/bash

set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

detect_tag $REPO_ROOT/dist/.tag

IMG=haproxy
TAG=1.8.5-$TAG

build() {
	pushd $(dirname "${BASH_SOURCE}")
	cp $REPO_ROOT/dist/voyager/voyager-alpine-amd64 voyager
	chmod +x voyager
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	rm voyager
	popd
}

binary_repo $@
