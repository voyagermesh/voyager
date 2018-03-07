#!/bin/bash

set -x -e

wget https://cdn.rawgit.com/Mirantis/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.9.sh
chmod +x dind-cluster-v1.9.sh
./dind-cluster-v1.9.sh up

kubectl get nodes
