#!/usr/bin/env bash

set -eoux pipefail

ORG_NAME=appscode
REPO_NAME=voyager
OPERATOR_NAME=voyager
APP_LABEL=voyager #required for `kubectl describe deploy -n kube-system -l app=$APP_LABEL`

export APPSCODE_ENV=dev
export DOCKER_REGISTRY=appscodeci

# get concourse-common
pushd $REPO_NAME
git status # required, otherwise you'll get error `Working tree has modifications.  Cannot add.`. why?
git subtree pull --prefix hack/libbuild https://github.com/appscodelabs/libbuild.git master --squash -m 'concourse'
popd

source $REPO_NAME/hack/libbuild/concourse/init.sh

cp creds/gcs.json /gcs.json
cp creds/voyager/.env $GOPATH/src/github.com/$ORG_NAME/$REPO_NAME/hack/config/.env

pushd $GOPATH/src/github.com/$ORG_NAME/$REPO_NAME

# install dependencies
./hack/builddeps.sh

./hack/docker/voyager/setup.sh
./hack/docker/haproxy/1.8.12-alpine/setup.sh

./hack/docker/voyager/setup.sh push
./hack/docker/haproxy/1.8.12-alpine/setup.sh push

./hack/deploy/voyager.sh --provider=$ClusterProvider

./hack/make.py test e2e --cloud-provider=$ClusterProvider --selfhosted-operator

popd
