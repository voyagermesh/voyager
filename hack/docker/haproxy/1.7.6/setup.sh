#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

detect_tag $REPO_ROOT/dist/.tag

IMG=haproxy
TAG=1.7.6-$TAG

build() {
	pushd $(dirname "${BASH_SOURCE}")
	wget -O kloader https://cdn.appscode.com/binaries/kloader/3.0.0/kloader-linux-amd64
	chmod +x kloader
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	rm kloader
	popd
}

binary_repo $@
