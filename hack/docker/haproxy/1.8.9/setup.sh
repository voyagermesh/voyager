#!/bin/bash

# Copyright AppsCode Inc. and Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eou pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../../..

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

detect_tag $REPO_ROOT/dist/.tag

IMG=haproxy
TAG=1.8.9-$TAG
DOCKER_REGISTRY=${DOCKER_REGISTRY:-appscode}

build() {
    pushd $(dirname "${BASH_SOURCE}")
    cp $REPO_ROOT/dist/voyager/voyager-linux-amd64 voyager
    chmod +x voyager

    # download socklog (`socklog` not available for `stretch`, use `jessie` deb instead)
    curl -L -o socklog.deb http://ftp.us.debian.org/debian/pool/main/s/socklog/socklog_2.1.0-8_amd64.deb
    # download auth-request.lua
    curl -fsSL -o auth-request.lua https://raw.githubusercontent.com/appscode/haproxy-auth-request/v1.8.9/auth-request.lua

    local cmd="docker build --pull -t $DOCKER_REGISTRY/$IMG:$TAG ."
    echo $cmd
    $cmd
    rm voyager socklog.deb auth-request.lua
    popd
}

binary_repo $@
