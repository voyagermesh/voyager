#!/bin/sh

set -x -e

apk update
apk add python py-pip
pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage
pip install pyyaml

mv kubeconfig/kubeconfig-0.0.1/* /
ls /
./src/github.com/appscode/voyager/hack/make.py test minikube --kubeconfig=/kubeconfig
