# Uninstall Voyager
Please follow the steps below to uninstall Voyager:

1. Delete the deployment and service used for Voyager operator.
```console
$ ./hack/deploy/uninstall.sh 
+ kubectl delete deployment -l app=voyager -n kube-system
deployment "voyager-operator" deleted
+ kubectl delete service -l app=voyager -n kube-system
service "voyager-operator" deleted
+ kubectl delete serviceaccount -l app=voyager -n kube-system
No resources found
+ kubectl delete clusterrolebindings -l app=voyager -n kube-system
No resources found
+ kubectl delete clusterrole -l app=voyager -n kube-system
No resources found
```

2. Now, wait several seconds for Voyager to stop running. To confirm that Voyager operator pod(s) have stopped running, run:
```console
$ kubectl get pods --all-namespaces -l app=voyager
```

3. To keep a copy of your existing Voyager objects, run:
```console
$ kubectl get ingress.voyager.appscode.com --all-namespaces -o yaml > ingress.yaml
$ kubectl get certificate.voyager.appscode.com --all-namespaces -o yaml > certificate.yaml
```

4. To delete existing Voyager objects from all namespaces, run the following command in each namespace one by one.
```console
$ kubectl delete ingress.voyager.appscode.com --all --cascade=false
$ kubectl delete certificate.voyager.appscode.com --all --cascade=false
```

5. Delete the old TPR-registration.
```console
kubectl delete thirdpartyresource -l app=voyager
```
