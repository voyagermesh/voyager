#!/bin/bash

set -x
set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT="$GOPATH/src/github.com/appscode/voyager"
rm -rf $REPO_ROOT/dist

./hack/docker/voyager/setup.sh
env APPSCODE_ENV=prod ./hack/docker/voyager/setup.sh release

./hack/docker/haproxy/1.7.9/setup.sh
./hack/docker/haproxy/1.7.9/setup.sh release

rm $REPO_ROOT/dist/.tag
