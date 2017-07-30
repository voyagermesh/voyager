#!/bin/bash
set -x

kubectl delete deployment -l app=voyager -n kube-system
kubectl delete service -l app=voyager -n kube-system

# Delete RBAC objects, if --rbac flag was used.
kubectl delete serviceaccount -l app=voyager -n kube-system
kubectl delete clusterrolebindings -l app=voyager -n kube-system
kubectl delete clusterrole -l app=voyager -n kube-system
