#!/bin/bash

set -x -e

ROOT_DIR=$(pwd)
mv kubeconfig/kubeconfig-0.0.1/* /
mkdir -p $GOPATH/src/github.com/appscode
cp -r pull-request $GOPATH/src/github.com/appscode/voyager
cd $GOPATH/src/github.com/appscode/voyager/hack
git rev-parse HEAD
./make.py test minikube --kubeconfig=/kubeconfig &> $ROOT_DIR/test_result/message
