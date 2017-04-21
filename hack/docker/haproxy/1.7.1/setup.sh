#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

IMG=haproxy
TAG=1.7.1-k8s

build() {
	pushd $(dirname "${BASH_SOURCE}")
	gsutil cp gs://appscode-dev/binaries/reloader/0.3/reloader-linux-amd64 reloader
	chmod +x reloader
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	rm reloader
	popd
}

binary_repo $@
