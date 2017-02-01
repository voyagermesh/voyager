#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

LIB_ROOT=$(dirname "${BASH_SOURCE}")/../../../..
source "$LIB_ROOT/hack/libbuild/common/lib.sh"
source "$LIB_ROOT/hack/libbuild/common/public_image.sh"

IMG=haproxy
TAG=1.7.2-k8s

build() {
	pushd $(dirname "${BASH_SOURCE}")
	wget -O kloader https://cdn.appscode.com/binaries/kloader/1.5.0/kloader-linux-amd64
	chmod +x kloader
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	rm kloader
	popd
}

binary_repo $@
