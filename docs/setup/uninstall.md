---
title: Uninstall Voyager
description: Voyager Uninstall
menu:
  product_voyager_5.0.0-rc.10:
    identifier: uninstall-voyager
    name: Uninstall
    parent: setup
    weight: 20
product_name: voyager
menu_name: product_voyager_5.0.0-rc.10
section_menu_id: setup
---

# Uninstall Voyager

Please follow the steps below to uninstall Voyager:

- Delete the deployment and service used for Voyager operator.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.10/hack/deploy/uninstall.sh | bash

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

- Now, wait several seconds for Voyager to stop running. To confirm that Voyager operator pod(s) have stopped running, run:

```console
$ kubectl get pods --all-namespaces -l app=voyager
```

- To keep a copy of your existing Voyager objects, run:

```console
$ kubectl get ingress.voyager.appscode.com --all-namespaces -o yaml > ingress.yaml
$ kubectl get certificate.voyager.appscode.com --all-namespaces -o yaml > certificate.yaml
```

- To delete existing Voyager objects from all namespaces, run the following command in each namespace one by one.

```console
$ kubectl delete ingress.voyager.appscode.com --all --cascade=false
$ kubectl delete certificate.voyager.appscode.com --all --cascade=false
```

- Delete the old CRD-registration.

```console
kubectl delete crd -l app=voyager
```
