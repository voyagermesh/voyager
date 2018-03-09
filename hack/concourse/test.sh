#!/bin/sh

set -x -e

mv kubeconfig/kubeconfig-0.0.1/* /
cd src/github.com/appscode/voyager/hack
ls
./make.py test minikube --kubeconfig=/kubeconfig
