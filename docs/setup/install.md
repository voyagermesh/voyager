---
title: Install Voyager
description: Voyager Install
menu:
  product_voyager_6.0.0-alpha.0:
    identifier: install-voyager
    name: Install
    parent: setup
    weight: 10
product_name: voyager
menu_name: product_voyager_6.0.0-alpha.0
section_menu_id: setup
---

# Installation Guide

## Using YAML
Voyager can be installed via installer script included in the [/hack/deploy](https://github.com/appscode/voyager/tree/6.0.0-alpha.0/hack/deploy) folder.

```console
# provider=acs
# provider=aws
# provider=azure
# provider=baremetal
# provider=gce
# provider=gke
# provider=minikube
# provider=openstack

$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh | bash -s -- -h
voyager.sh - install voyager operator

voyager.sh [options]

options:
-h, --help                         show brief help
-n, --namespace=NAMESPACE          specify namespace (default: kube-system)
-p, --provider=PROVIDER            specify a cloud provider
    --rbac                         create RBAC roles and bindings
    --docker-registry              docker registry used to pull voyager images (default: appscode)
    --image-pull-secret            name of secret used to pull voyager operator images
    --restrict-to-namespace        restrict voyager to its own namespace
    --run-on-master                run voyager operator on master
    --enable-apiserver     configure admission webhook for Voyager CRDs
    --template-cfgmap=CONFIGMAP    name of configmap with custom templates

# install without RBAC roles
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider

# Install with RBAC roles
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --rbac
```

If you would like to run Voyager operator pod in `master` instances, pass the `--run-on-master` flag:

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --run-on-master [--rbac]
```

Voyager operator will be installed in a `kube-system` namespace by default. If you would like to run Voyager operator pod in `voyager` namespace, pass the `--namespace=voyager` flag:

```console
$ kubectl create namespace voyager
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --namespace=voyager [--run-on-master] [--rbac]
```

By default, Voyager operator will watch Ingress objects in any namespace. If you would like to restrict Voyager to Ingress and Services in its own namespace, pass the `--restrict-to-namespace` flag:

```console
$ kubectl create namespace voyager
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --restrict-to-namespace [--namespace=voyager] [--run-on-master] [--rbac]
```

If you are using a private Docker registry, you need to pull the following 2 docker images:

 - [appscode/voyager](https://hub.docker.com/r/appscode/voyager)
 - [appscode/haproxy](https://hub.docker.com/r/appscode/haproxy)

To pass the address of your private registry and optionally a image pull secret use flags `--docker-registry` and `--image-pull-secret` respectively.

```console
$ kubectl create namespace voyager
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --docker-registry=MY_REGISTRY [--image-pull-secret=SECRET_NAME] [--rbac]
```

Voyager implements a [validating admission webhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19) to validate Voyager CRDs. To enable this feature, pass the `--enable-apiserver` flag. _Please note that, this works with Kubernetes 1.9.0 or later releases_.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/6.0.0-alpha.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --enable-apiserver [--rbac]
```

__N.B.__
- If you are installing Voyager on a GKE cluster, you will need cluster admin permissions to install Voyager operator. Run the following command to grant admin permision to the cluster.

```console
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=google-email-for-gce-project
```


## Using Helm
Voyager can be installed via [Helm](https://helm.sh/) using the [chart](https://github.com/appscode/voyager/tree/6.0.0-alpha.0/chart/stable/voyager) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```console
$ helm repo update
$ helm install stable/voyager --name my-release
```
To see the detailed configuration options, visit [here](https://github.com/appscode/voyager/tree/6.0.0-alpha.0/chart/stable/voyager).


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

Version = 6.0.0-alpha.0
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = ab0b38d8f5d5b4b4508768a594a9d98f2c76abd8
GitBranch = release-4.0
GitTag = 6.0.0-alpha.0
CommitTimestamp = 2017-10-08T12:45:26
```
