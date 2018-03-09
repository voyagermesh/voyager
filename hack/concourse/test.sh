#!/bin/sh

set -x -e

mv kubeconfig/kubeconfig-0.0.1/* /
mkdir -p $GOPATH/src/github.com/appscode
cp -r voyager $GOPATH/src/github.com/appscode
cd $GOPATH/src/github.com/appscode/voyager/hack
./make.py test minikube --kubeconfig=/kubeconfig
