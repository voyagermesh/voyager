#!/bin/bash

export CLOUD_PROVIDER=openstack
export CLOUD_CONFIG=
export INGRESS_CLASS=

if [ $# -eq 0 ]; then
    cat ./hack/deploy/without-rbac.yaml | envsubst | kubectl apply -f -
elif [ $1 == '--rbac' ]; then
    cat ./hack/deploy/with-rbac.yaml | envsubst | kubectl apply -f -
else
    echo 'Usage: ./hack/deploy/aws.sh [--rbac]'
fi
