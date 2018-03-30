#!/bin/bash

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

function cleanup {
    pharmer get cluster
    pharmer delete cluster $NAME
    pharmer get cluster
    sleep 600
    pharmer apply $NAME
    pharmer get cluster

#    for debugging cluster
#    for f in $(find ~/.pharmer -type f)
#    do
#        echo ------$f------; cat $f; echo; echo;
#    done
}
trap cleanup EXIT

pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc3 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
kubectl get nodes

#for debugging cluster
#for f in $(find ~/.pharmer -type f)
#do
#    echo ------$f------; cat $f; echo; echo;
#done

./make.py test baremetal
