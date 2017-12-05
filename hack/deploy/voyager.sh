#!/bin/bash

provider=$1

case "$provider" in
	acs)
		export CLOUD_PROVIDER=acs
		export CLOUD_CONFIG=/etc/kubernetes/azure.json
		export INGRESS_CLASS=
		;;
	aws)
		export CLOUD_PROVIDER=aws
		export CLOUD_CONFIG=
		export INGRESS_CLASS=
		;;
	azure)
		export CLOUD_PROVIDER=azure
		export CLOUD_CONFIG=/etc/kubernetes/azure.json
		export INGRESS_CLASS=
		;;
	baremetal)
		export CLOUD_PROVIDER=
		export CLOUD_CONFIG=
		;;
	gce)
		export CLOUD_PROVIDER=gce
		export CLOUD_CONFIG=
		export INGRESS_CLASS=
		;;
	gke)
		export CLOUD_PROVIDER=gke
		export CLOUD_CONFIG=
		export INGRESS_CLASS=voyager
		;;
	minikube)
		export CLOUD_PROVIDER=minikube
		export CLOUD_CONFIG=
		export INGRESS_CLASS=
		;;
	openstack)
		export CLOUD_PROVIDER=openstack
		export CLOUD_CONFIG=
		export INGRESS_CLASS=
		;;
	*)
		echo 'Usage: ./hack/deploy/voyager.sh $provider [--rbac]'
		exit 1
		;;
esac

shift

if [ $# -eq 0 ]; then
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.6/hack/deploy/without-rbac.yaml | envsubst | kubectl apply -f -
elif [ $1 == '--rbac' ]; then
    curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.6/hack/deploy/with-rbac.yaml | envsubst | kubectl apply -f -
else
    echo 'Usage: ./hack/deploy/voyager.sh $provider [--rbac]'
fi
