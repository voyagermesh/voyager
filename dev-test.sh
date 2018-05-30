#!/bin/bash
set -xeou pipefail

# uses: $ ./dev-test.sh --provider=minikube --docker-registry=appscodeci

export APPSCODE_ENV=dev

while test $# -gt 0; do
    case "$1" in
        --provider*)
            provider=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --docker-registry*)
            export DOCKER_REGISTRY=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
    esac
done

# build & push voyager docker image
./hack/docker/voyager/setup.sh
./hack/docker/voyager/setup.sh push

# build & push haproxy docker image
./hack/docker/haproxy/1.8.8-alpine/setup.sh
./hack/docker/haproxy/1.8.8-alpine/setup.sh push

# deploy voyager operator
./hack/deploy/voyager.sh --provider=${provider}

# run e2e tests
./hack/make.py test e2e --cloud-provider=${provider} --selfhosted-operator

# uninstall voyager operator
./hack/deploy/voyager.sh --provider=${provider} --uninstall
