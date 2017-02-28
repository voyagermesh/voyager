#!/usr/bin/env bash

set -o nounset
set -o pipefail

RETVAL=0
ROOT=$PWD

hard() {
    minikube delete
    minikube start
}

soft() {
    kubectl delete ingress.appscode.com base-ingress
    kubectl delete ingress.appscode.com base-d-ingress
    kubectl delete ingress base-ingress
    kubectl delete rc/voyager-base-ingress svc/voyager-base-ingress configmap/voyager-base-ingress daemonset/voyager-base-d-ingress svc/voyager-base-d-ingress configmap/voyager-base-d-ingress
}

if [ $# -eq 0 ]; then
	soft
	exit $RETVAL
fi

case "$1" in
  hard)
      hard
      ;;
  soft)
      soft
	  ;;
  *)  echo $"Usage: $0 {soft|hard}"
      RETVAL=1
      ;;
esac
exit $RETVAL