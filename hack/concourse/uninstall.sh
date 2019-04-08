#!/usr/bin/env bash

set -x

# uninstall operator
./hack/deploy/voyager.sh --uninstall --purge

# remove creds
rm -rf /gcs.json
rm -rf hack/config/.env

# remove docker images
source "hack/libbuild/common/lib.sh"
detect_tag ''

# delete docker image on exit
./hack/libbuild/docker.py del_tag $DOCKER_REGISTRY voyager $TAG
./hack/libbuild/docker.py del_tag $DOCKER_REGISTRY haproxy 1.9.6-$TAG-alpine
