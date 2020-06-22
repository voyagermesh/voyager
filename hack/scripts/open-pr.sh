#!/bin/bash

# Copyright The KubeVault Authors.
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

SCRIPT_ROOT=$(realpath $(dirname "${BASH_SOURCE[0]}")/../..)
SCRIPT_NAME=$(basename "${BASH_SOURCE[0]}")
pushd $SCRIPT_ROOT

# http://redsymbol.net/articles/bash-exit-traps/
function cleanup() {
  popd
}
trap cleanup EXIT

git add --all
if git diff -s --exit-code HEAD; then
	echo "Docs are already up-to-date!"
	exit 0
fi

pr_branch=${GITHUB_REPOSITORY}@${GITHUB_SHA:0:8}
git checkout -b $pr_branch
git commit -a -s -m "Update docs for $pr_branch"
git push -u origin HEAD
hub pull-request \
    --labels automerge \
    --message "Update docs for $pr_branch" \
    --message "$(git show -s --format=%b)"
