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

export HAPROXY_CONTROLLER_ARGS="$@"
export >/etc/envvars

[[ $DEBUG == true ]] && set -x

# create haproxy.cfg dir
mkdir /etc/haproxy
touch /var/run/haproxy.pid
mkdir -p /etc/ssl/private/haproxy

echo "Starting runit..."
exec /sbin/runsvdir -P /etc/service
