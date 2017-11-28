---
title: Uninstall | Voyager
description: Voyager Uninstall
menu:
  product_voyager_5.0.0-rc.3:
    identifier: uninstall-voyager
    name: Uninstall
    parent: getting-started
    weight: 50
product_name: voyager
left_menu: product_voyager_5.0.0-rc.3
section_menu_id: getting-started
url: /products/voyager/5.0.0-rc.3/getting-started/uninstall/
aliases:
  - /products/voyager/5.0.0-rc.3/uninstall/
---

# Uninstall Voyager
Please follow the steps below to uninstall Voyager:

1. Delete the deployment and service used for Voyager operator.
```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.4/hack/deploy/uninstall.sh | bash

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

5. Delete the old CRD-registration.
```console
kubectl delete crd -l app=voyager
```
