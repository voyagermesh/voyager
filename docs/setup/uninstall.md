---
title: Uninstall Voyager
description: Voyager Uninstall
menu:
  product_voyager_7.0.0:
    identifier: uninstall-voyager
    name: Uninstall
    parent: setup
    weight: 20
product_name: voyager
menu_name: product_voyager_7.0.0
section_menu_id: setup
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Uninstall Voyager

To uninstall Voyager operator, run the following command:

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.0.0/hack/deploy/voyager.sh \
    | bash -s -- --uninstall [--namespace=NAMESPACE]

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

The above command will leave the Voyager crd objects as-is. If you wish to **nuke** all Voyager crd objects, also pass the `--purge` flag. This will keep a copy of Voyager crd objects in your current directory.
