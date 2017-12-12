#!/bin/bash

kubectl run nginx --image=nginx
kubectl expose deployment nginx --name=web --port=80 --target-port=80

kubectl run echoserver --image=gcr.io/google_containers/echoserver:1.4
kubectl expose deployment echoserver --name=echo --port=80 --target-port=8080
