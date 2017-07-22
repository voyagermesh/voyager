#!/bin/bash

export CLOUD_PROVIDER=azure
export CLOUD_CONFIG=/etc/kubernetes/azure.json
export INGRESS_CLASS=

if [ $# -eq 0 ]; then
    cat ./hack/deploy/without-rbac.yaml | envsubst | kubectl apply -f -
elif [ $1 == '--rbac' ]; then
    cat ./hack/deploy/with-rbac.yaml | envsubst | kubectl apply -f -
else
    echo 'Usage: ./hack/deploy/azure.sh [--rbac]'
fi
