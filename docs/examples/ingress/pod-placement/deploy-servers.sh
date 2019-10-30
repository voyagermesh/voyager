#!/bin/bash

# Copyright The Voyager Authors.
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

set -x

kubectl create namespace demo

kubectl run nginx --image=nginx --namespace=demo
kubectl expose deployment nginx --name=web --namespace=demo --port=80 --target-port=80

kubectl run echoserver --image=gcr.io/google_containers/echoserver:1.4 --namespace=demo
kubectl expose deployment echoserver --name=rest --namespace=demo --port=80 --target-port=8080
