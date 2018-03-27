#!/bin/bash
set -xeou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/appscode/voyager"

export APPSCODE_ENV=prod

pushd $REPO_ROOT

rm -rf dist

./hack/docker/voyager/setup.sh
./hack/docker/voyager/setup.sh release

./hack/docker/haproxy/1.8.5/setup.sh
./hack/docker/haproxy/1.8.5/setup.sh release

rm dist/.tag

popd
