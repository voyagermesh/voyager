#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/appscode/voyager"

export APPSCODE_ENV=prod

pushd $REPO_ROOT

rm -rf dist

./hack/make.py build voyager

./hack/docker/haproxy/1.9.6/setup.sh
./hack/docker/haproxy/1.9.6/setup.sh release

./hack/docker/haproxy/1.9.6-alpine/setup.sh
./hack/docker/haproxy/1.9.6-alpine/setup.sh release

rm dist/.tag

popd
