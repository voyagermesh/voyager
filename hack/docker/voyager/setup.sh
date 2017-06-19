#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
ROOT=$GOPATH
REPO_ROOT=$GOPATH/src/github.com/appscode/voyager

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=voyager

mkdir -p $REPO_ROOT/dist
if [ -f "$REPO_ROOT/dist/.tag" ]; then
	export $(cat $REPO_ROOT/dist/.tag | xargs)
fi

clean() {
    pushd $REPO_ROOT/hack/docker/voyager
	rm -rf voyager
	popd
}

build_binary() {
	pushd $REPO_ROOT
	./hack/builddeps.sh
    ./hack/make.py build voyager
	detect_tag $REPO_ROOT/dist/.tag
	popd
}

build_docker() {
	pushd $REPO_ROOT/hack/docker/voyager
	cp $REPO_ROOT/dist/voyager/voyager-linux-amd64 voyager
	chmod 755 voyager

	cat >Dockerfile <<EOL
FROM alpine

RUN set -x \
  && apk update \
  && apk add ca-certificates \
  && rm -rf /var/cache/apk/*

COPY voyager /voyager
USER nobody:nobody
ENTRYPOINT ["/voyager"]
EOL
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd

	rm voyager Dockerfile
	popd
}

build() {
	build_binary
	build_docker
}

docker_push() {
    if [ "$APPSCODE_ENV" = "prod" ]; then
        echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?"
        exit 1
    fi
    if [ "$TAG_STRATEGY" = "git_tag" ]; then
        echo "Are you trying to 'release' binaries to prod?"
        exit 1
    fi
    hub_canary
}

docker_release() {
    if [ "$APPSCODE_ENV" != "prod" ]; then
        echo "'release' only works in PROD env."
        exit 1
    fi
    if [ "$TAG_STRATEGY" != "git_tag" ]; then
        echo "'apply_tag' to release binaries and/or docker images."
        exit 1
    fi
    hub_up
}

source_repo $@
