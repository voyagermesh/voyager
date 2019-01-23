#!/bin/bash
set -eou pipefail

# usages:
# $ ./dev-test.sh --provider=minikube --docker-registry=appscodeci build|push|install|e2e|uninstall
# $ ./dev-test.sh --provider=minikube --docker-registry=appscodeci

pushd ${GOPATH}/src/github.com/appscode/voyager

export APPSCODE_ENV=dev

while test $# -gt 0; do
  case "$1" in
    --provider*)
      provider=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --docker-registry*)
      export DOCKER_REGISTRY=$(echo $1 | sed -e 's/^[^=]*=//g')
      shift
      ;;
    --*)
      echo "Error: Unknown option ($1)"
      exit 1
      ;;
    *)
      break
      ;;
  esac
done

docker_build() {
  echo "===building voyager docker image==="
  ./hack/docker/voyager/setup.sh
  echo "===building haproxy docker image==="
  ./hack/docker/haproxy/1.9.2-alpine/setup.sh
}

docker_push() {
  echo "===pushing voyager docker image==="
  ./hack/docker/voyager/setup.sh push
  echo "===pushing haproxy docker image==="
  ./hack/docker/haproxy/1.9.2-alpine/setup.sh push
}

install() {
  echo "===installing voyager operator==="
  ./hack/deploy/voyager.sh --provider=${provider}
}

e2e() {
  echo "===running voyager e2e tests==="
  ./hack/make.py test e2e --cloud-provider=${provider} --selfhosted-operator
}

uninstall() {
  echo "===uninstalling voyager operator==="
  ./hack/deploy/voyager.sh --provider=${provider} --uninstall --purge
}

if test $# -gt 0; then
  case "$1" in
    "build")
      docker_build
      exit 0
      ;;
    "push")
      docker_push
      exit 0
      ;;
    "install")
      install
      exit 0
      ;;
    "e2e")
      e2e
      exit 0
      ;;
    "uninstall")
      uninstall
      exit 0
      ;;
    *)
      echo "Error: Command not supported ($1)"
      exit 1
      ;;
  esac
fi

docker_build
docker_push
install
e2e
uninstall

popd
