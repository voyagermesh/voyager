#!/bin/bash

set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

detect_tag $REPO_ROOT/dist/.tag

IMG=haproxy
TAG=1.8.8-$TAG

build() {
	pushd $(dirname "${BASH_SOURCE}")
	cp $REPO_ROOT/dist/voyager/voyager-linux-amd64 voyager
	chmod +x voyager
	# download socklog (`socklog` not available for `stretch`, use `jessie` deb instead)
	curl -o socklog.deb http://ftp.us.debian.org/debian/pool/main/s/socklog/socklog_2.1.0-8_amd64.deb
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	rm voyager socklog.deb
	popd
}

binary_repo $@
