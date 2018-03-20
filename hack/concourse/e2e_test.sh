#!/bin/sh

set -x -e

mkdir -p $GOPATH/src/github.com/appscode
cp -r voyager $GOPATH/src/github.com/appscode
cd $GOPATH/src/github.com/appscode/voyager/hack

NAME=voyager$(git rev-parse HEAD) #name of the cluster

cat > cred.json <<EOF
{
	"token" : "$TOKEN"
}
EOF

pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc3 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
kubectl get nodes

./make.py test minikube

pharmer delete cluster $NAME
