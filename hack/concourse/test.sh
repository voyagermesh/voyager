#!/bin/bash

set -x -e

ls
apk update
apk add curl bash
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
 chmod +x ./kubectl
mv ./kubectl /usr/local/bin/kubectl

curl -LO https://cdn.rawgit.com/Mirantis/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.9.sh
chmod +x dind-cluster-v1.9.sh
./dind-cluster-v1.9.sh up

kubectl get nodes
