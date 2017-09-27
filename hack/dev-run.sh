#!/usr/bin/env bash

REPOROOT="$GOPATH/src/github.com/appscode/voyager"

tls-mounter() {
    echo "Running tls-mounter for dev mode"
    "${REPOROOT}"/hack/make.py

    mkdir -p /tmp/tls-mount

    kubectl apply -f "${REPOROOT}"/apis/voyager/v1beta1/crds.yaml
    kubectl create secret tls test-secret --cert="${REPOROOT}"/test/testdata/certs/ca.crt --key="${REPOROOT}"/test/testdata/certs/ca.key

    cat <<EOF | kubectl apply -f -
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  tls:
  - secretName: test-secret
    hosts:
    - appscode.example.com
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: s1
          servicePort: "80"
        path: /foo
      - backend:
          serviceName: s2
          servicePort: "80"
        path: /bar
EOF

    voyager tls-mounter \
      --ingress-api-version=voyager.appscode.com/v1beta1 \
      --ingress-name=test-ingress \
      --cloud-provider=minikube \
      --v=3 \
      --kubeconfig="${HOME}"/.kube/config \
      --mount=/tmp/tls-mount
}

RETVAL=0
if [ $# -eq 0 ]; then
    echo "No Target specified"
    exit 1
fi

case "$1" in
    tls-mounter)
        tls-mounter
        ;;
    *)	(10)
        echo $"Usage: $0 {tls-mounter}"
        RETVAL=1
esac
exit ${RETVAL}