---
title: Install | Voyager
description: Voyager Install
menu:
  product_voyager_5.0.0-rc.3:
    identifier: install-voyager
    name: Install
    parent: getting-started
    weight: 35
product_name: voyager
left_menu: product_voyager_5.0.0-rc.3
section_menu_id: getting-started
url: /products/voyager/5.0.0-rc.3/getting-started/install/
aliases:
  - /products/voyager/5.0.0-rc.3/install/
---

# Installation Guide

## Using YAML
Voyager can be installed via cloud provider specific YAML files included in the [/hack/deploy](https://github.com/appscode/voyager/tree/5.0.0-rc.4/hack) folder. To use in a RBAC enabled cluster, pass the `--rbac` flag.

```console
# provider=acs
# provider=aws
# provider=azure
# provider=baremetal
# provider=gce
# provider=gke
# provider=minikube
# provider=openstack

# Install without RBAC roles
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.4/hack/deploy/voyager.sh \
    | bash -s -- "$provider"

# Install with RBAC roles
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.4/hack/deploy/voyager.sh \
    | bash -s -- "$provider" --rbac
```

If you would like to run Voyager operator pod in `master` instances, apply the following patch:

```console
$ kubectl patch deploy voyager-operator -n kube-system \
    --patch "$(curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.4/hack/deploy/run-on-master.yaml)"
```


## Using Helm
Voyager can be installed via [Helm](https://helm.sh/) using the [chart](/chart/stable/voyager) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```console
$ helm repo update
$ helm install stable/voyager --name my-release
```
To see the detailed configuration options, visit [here](/chart/stable/voyager/README.md).


## Verify installation
To check if Voyager operator pods have started, run the following command:
```console
$ kubectl get pods --all-namespaces -l app=voyager --watch
```

Once the operator pods are running, you can cancel the above command by typing `Ctrl+C`.

Now, to confirm CRD groups have been registered by the operator, run the following command:
```console
$ kubectl get crd -l app=voyager
```

Now, you are ready to create your first ingress using Voyager.

## Using kubectl
Since Voyager uses its own TPR/CRD, you need to use full resource kind to find it with kubectl.
```console
# List all voyager ingress
$ kubectl get ingress.voyager.appscode.com --all-namespaces

# List voyager ingress for a namespace
$ kubectl get ingress.voyager.appscode.com -n <namespace>

# Get Ingress YAML
$ kubectl get ingress.voyager.appscode.com -n <namespace> <ingress-name> -o yaml

# Describe Ingress. Very useful to debug problems.
$ kubectl describe ingress.voyager.appscode.com -n <namespace> <ingress-name>
```

## Detect Voyager version
To detect Voyager version, exec into the operator pod and run `voyager version` command.
```console
$ POD_NAMESPACE=kube-system
$ POD_NAME=$(kubectl get pods -n $POD_NAMESPACE -l app=voyager -o jsonpath={.items[0].metadata.name})
$ kubectl exec -it $POD_NAME -n $POD_NAMESPACE voyager version

Version = 5.0.0-rc.4
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = ab0b38d8f5d5b4b4508768a594a9d98f2c76abd8
GitBranch = release-4.0
GitTag = 5.0.0-rc.4
CommitTimestamp = 2017-10-08T12:45:26
```
