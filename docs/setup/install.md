---
title: Install Voyager
description: Voyager Install
menu:
  product_voyager_7.2.0:
    identifier: install-voyager
    name: Install
    parent: setup
    weight: 10
product_name: voyager
menu_name: product_voyager_7.2.0
section_menu_id: setup
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Installation Guide

Voyager operator can be installed via a script or as a Helm chart.

<ul class="nav nav-tabs" id="installerTab" role="tablist">
  <li class="nav-item">
    <a class="nav-link active" id="script-tab" data-toggle="tab" href="#script" role="tab" aria-controls="script" aria-selected="true">Script</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="helm-tab" data-toggle="tab" href="#helm" role="tab" aria-controls="helm" aria-selected="false">Helm</a>
  </li>
</ul>
<div class="tab-content" id="installerTabContent">
  <div class="tab-pane fade show active" id="script" role="tabpanel" aria-labelledby="script-tab">

## Using Script

To install Voyager in your Kubernetes cluster, pick the appropriate cluster provider and run the following command:

```console
# provider=acs
# provider=aks
# provider=aws
# provider=azure
# provider=baremetal
# provider=gce
# provider=gke
# provider=minikube
# provider=openstack
# provider=metallb
# provider=digitalocean
# provider=linode

$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider
```

After successful installation, you should have a `voyager-operator-***` pod running in the `kube-system` namespace.

```console
$ kubectl get pods -n kube-system | grep voyager-operator
voyager-operator-846d47f489-jrb58       1/1       Running   0          48s
```

#### Customizing Installer

The installer script and associated yaml files can be found in the [/hack/deploy](https://github.com/appscode/voyager/tree/7.2.0/hack/deploy) folder. To see the full list of flags available to installer, use the `-h` flag.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh | bash -s -- -h
voyager.sh - install voyager operator

voyager.sh [options]

options:
-h, --help                         show brief help
-n, --namespace=NAMESPACE          specify namespace (default: kube-system)
-p, --provider=PROVIDER            specify a cloud provider
    --rbac                         create RBAC roles and bindings (default: true)
    --docker-registry              docker registry used to pull voyager images (default: appscode)
    --haproxy-image-tag            tag of Docker image containing HAProxy binary (default: 1.8.9-7.2.0-alpine)
    --image-pull-secret            name of secret used to pull voyager operator images
    --restrict-to-namespace        restrict voyager to its own namespace
    --run-on-master                run voyager operator on master
    --enable-validating-webhook    enable/disable validating webhooks for voyager CRDs
    --template-cfgmap=CONFIGMAP    name of configmap with custom templates
    --enable-analytics             send usage events to Google Analytics (default: true)
    --uninstall                    uninstall voyager
    --purge                        purges Voyager crd objects and crds
```

If you would like to run Voyager operator pod in `master` instances, pass the `--run-on-master` flag:

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --run-on-master [--rbac]
```

Voyager operator will be installed in a `kube-system` namespace by default. If you would like to run Voyager operator pod in `voyager` namespace, pass the `--namespace=voyager` flag:

```console
$ kubectl create namespace voyager
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --namespace=voyager [--run-on-master] [--rbac]
```

By default, Voyager operator will watch Ingress objects in any namespace. If you would like to restrict Voyager to Ingress and Services in its own namespace, pass the `--restrict-to-namespace` flag:

```console
$ kubectl create namespace voyager
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --namespace=voyager --restrict-to-namespace [--run-on-master] [--rbac]
```

If you are using a private Docker registry, you need to pull the following 2 docker images:

 - [appscode/voyager](https://hub.docker.com/r/appscode/voyager)
 - [appscode/haproxy](https://hub.docker.com/r/appscode/haproxy)

To pass the address of your private registry and optionally a image pull secret use flags `--docker-registry` and `--image-pull-secret` respectively.

```console
$ kubectl create namespace voyager
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --docker-registry=MY_REGISTRY [--image-pull-secret=SECRET_NAME] [--rbac]
```

By default, Voyager uses Alpine based HAProxy image (1.8.9-7.2.0-alpine). But you can also Debian based image for HAProxy by setting --haproxy-image-tag=1.8.9-7.2.0 flag.

Voyager implements a [validating admission webhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19) to validate Voyager CRDs. This is enabled by default for Kubernetes 1.9.0 or later releases. To disable this feature, pass the `--enable-validating-webhook=false` flag.

```console
$ curl -fsSL https://raw.githubusercontent.com/appscode/voyager/7.2.0/hack/deploy/voyager.sh \
    | bash -s -- --provider=$provider --enable-validating-webhook [--rbac]
```

To use custom templates to render HAProxy configuration, visit [here](/docs/guides/ingress/configuration/custom-templates.md).

</div>
<div class="tab-pane fade" id="helm" role="tabpanel" aria-labelledby="helm-tab">

## Using Helm

Voyager can be installed via [Helm](https://helm.sh/) 2.9.x or later versions using the [chart](https://github.com/appscode/voyager/tree/7.2.0/chart/voyager) from [AppsCode Charts Repository](https://github.com/appscode/charts). To install the chart with the release name `my-release`:

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm search appscode/voyager
NAME              CHART VERSION APP VERSION DESCRIPTION
appscode/voyager  7.2.0    7.2.0  Voyager by AppsCode - Secure HAProxy Ingress Controller...

# Kubernetes 1.8.x
$ helm install appscode/voyager --name voyager-operator --version 7.2.0 \
  --namespace kube-system \
  --set cloudProvider=$provider

# Kubernetes 1.9.0 or later
$ helm install appscode/voyager --name voyager-operator --version 7.2.0 \
  --namespace kube-system \
  --set cloudProvider=$provider \
  --set apiserver.ca="$(onessl get kube-ca)" \
  --set apiserver.enableValidatingWebhook=true
```

To install `onessl`, run the following commands:

```console
# Mac OSX amd64:
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-darwin-amd64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/

# Linux amd64:
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/

# Linux arm64:
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-arm64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/
```

To see the detailed configuration options, visit [here](https://github.com/appscode/voyager/tree/7.2.0/chart/voyager).

</div>

### Installing in GKE Cluster

If you are installing Voyager on a GKE cluster, you will need cluster admin permissions to install Voyager operator. Run the following command to grant admin permision to the cluster.

```console
# get current google identity
$ gcloud info | grep Account
Account: [user@example.org]

$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=user@example.org
```

### Installing in Minikube

Voyager can be used in minikube using `--provider=minikube`. In Minikube, a `LoadBalancer` type ingress will only assigned a NodePort.

### Installing in Baremetal Cluster

Voyager works great in baremetal cluster. To install, set `--provider=baremetal`. In baremetal cluster, `LoadBalancer` type ingress in not supported. You can use [NodePort](/docs/concepts/ingress-types/nodeport.md), [HostPort](/docs/concepts/ingress-types/hostport.md) or [Internal](/docs/concepts/ingress-types/internal.md) ingress objects.

### Installing in Baremetal Cluster with MetalLB

Follow the instructions for installing on baremetal cluster but specify `metallb` as provider. Then install MetalLB following the instructions [here](https://metallb.universe.tf/installation/). Now, you can use `LoadBalancer` type ingress in baremetal clusters.

### Installing in DigitalOcean Cluster

To use `LoadBalancer` type ingress in [DigitalOcean](https://www.digitalocean.com/) cluster, install Kubernetes [cloud controller manager for DigitalOcean](https://github.com/digitalocean/digitalocean-cloud-controller-manager). Otherwise set cloud provider to `barematal`.

### Installing in Linode Cluster

To use `LoadBalancer` type ingress in [Linode](https://www.linode.com/) cluster, install Kubernetes [cloud controller manager for Linode](https://github.com/pharmer/cloud-controller-manager). Otherwise set cloud provider to `barematal`.

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


## Configuring RBAC
Voyager creates two CRDs: `Ingress` and `Certificate`. Voyager installer will create 2 user facing cluster roles:

| ClusterRole           | Aggregates To | Desription                            |
|-----------------------|---------------|---------------------------------------|
| appscode:voyager:edit | admin, edit   | Allows edit access to Voyager CRDs, intended to be granted within a namespace using a RoleBinding. |
| appscode:voyager:view | view          | Allows read-only access to Voyager CRDs, intended to be granted within a namespace using a RoleBinding. |

These user facing roles supports [ClusterRole Aggregation](https://kubernetes.io/docs/admin/authorization/rbac/#aggregated-clusterroles) feature in Kubernetes 1.9 or later clusters.


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

Version = 7.2.0
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = ab0b38d8f5d5b4b4508768a594a9d98f2c76abd8
GitBranch = release-4.0
GitTag = 7.2.0
CommitTimestamp = 2017-10-08T12:45:26
```
