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

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

export APPSCODE_ENV=prod

pushd $REPO_ROOT

rm -rf dist

./hack/make.py build voyager

./hack/docker/haproxy/1.9.15/setup.sh
./hack/docker/haproxy/1.9.15/setup.sh release

./hack/docker/haproxy/1.9.15-alpine/setup.sh
./hack/docker/haproxy/1.9.15-alpine/setup.sh release

rm dist/.tag

popd
