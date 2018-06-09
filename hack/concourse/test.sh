#!/bin/bash

set -x -e

# start docker and log-in to docker-hub
entrypoint.sh
docker login --username=$DOCKER_USER --password=$DOCKER_PASS
docker run hello-world

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &> /dev/null
chmod +x ./kubectl
mv ./kubectl /bin/kubectl

# install pharmer
curl -LO https://cdn.appscode.com/binaries/pharmer/0.1.0-rc.4/pharmer-linux-amd64
chmod +x pharmer-linux-amd64
mv pharmer-linux-amd64 /bin/pharmer

function cleanup_test_stuff {
    set +e

    # delete cluster on exit
    pharmer get cluster
    pharmer delete cluster $NAME
    pharmer get cluster
    sleep 300
    pharmer apply $NAME
    pharmer apply $NAME
    pharmer get cluster

    # delete docker image on exit
    curl -LO https://raw.githubusercontent.com/appscodelabs/libbuild/master/docker.py
    chmod +x docker.py
    ./docker.py del_tag appscodeci voyager $VOYAGER_IMAGE_TAG
    ./docker.py del_tag appscodeci haproxy $HAPROXY_IMAGE_TAG
}
trap cleanup_test_stuff EXIT

# copy voyager to $GOPATH
mkdir -p $GOPATH/src/github.com/appscode
cp -r voyager $GOPATH/src/github.com/appscode

pushd $GOPATH/src/github.com/appscode/voyager

# name of the cluster
NAME=voyager$(git rev-parse --short HEAD)

# install dependencies
./hack/builddeps.sh

# build and push docker images
export APPSCODE_ENV=dev
export DOCKER_REGISTRY=appscodeci

# build & push voyager docker image
./hack/docker/voyager/setup.sh
./hack/docker/voyager/setup.sh push

# build & push haproxy docker image
./hack/docker/haproxy/1.8.8-alpine/setup.sh
./hack/docker/haproxy/1.8.8-alpine/setup.sh push

popd

cat > cred.json <<EOF
{
	"token" : "$TOKEN"
}
EOF

# create cluster using pharmer
pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc1 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.10.0
pharmer apply $NAME || true
pharmer apply $NAME
pharmer use cluster $NAME
#wait for cluster to be ready
sleep 120
kubectl get nodes

# create storageclass
cat > sc.yaml <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
parameters:
  zone: nyc1
provisioner: external/pharmer
EOF

# create storage-class
kubectl create -f sc.yaml
sleep 120
kubectl get storageclass

export CRED_DIR=$(pwd)/creds/voyager/gce.json

pushd $GOPATH/src/github.com/appscode/voyager

# create config/.env file that have all necessary creds
cat > hack/configs/.env <<EOF
TEST_GCE_PROJECT=$TEST_GCE_PROJECT
TEST_GCE_SERVICE_ACCOUNT_DATA=$CRED_DIR
TEST_ACME_USER_EMAIL=$TEST_ACME_USER_EMAIL
TEST_DNS_DOMAINS=$TEST_DNS_DOMAINS
EOF

# deploy voyager operator
source ./hack/deploy/voyager.sh --provider=baremetal
./hack/make.py test e2e --cloud-provider=baremetal --selfhosted-operator
