#!/bin/sh

set -x -e

docker run hello-world

source /docker-lib.sh
start_docker


apk --no-cache add curl bash

curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
 chmod +x ./kubectl
mv ./kubectl /usr/local/bin/kubectl

curl -LO https://cdn.rawgit.com/Mirantis/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.8.sh
chmod +x dind-cluster-v1.8.sh
./dind-cluster-v1.8.sh up


kubectl get nodes
