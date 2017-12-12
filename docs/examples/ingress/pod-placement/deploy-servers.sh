#!/bin/bash
set -x

kubectl create namespace demo

kubectl run nginx --image=nginx --namespace=demo
kubectl expose deployment nginx --name=web --namespace=demo --port=80 --target-port=80

kubectl run echoserver --image=gcr.io/google_containers/echoserver:1.4 --namespace=demo
kubectl expose deployment echoserver --name=rest --namespace=demo --port=80 --target-port=8080
