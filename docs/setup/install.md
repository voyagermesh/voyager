---
title: Install Voyager
description: Voyager Install
menu:
  product_voyager_{{ .version }}:
    identifier: install-voyager
    name: Install
    parent: setup
    weight: 10
product_name: voyager
menu_name: product_voyager_{{ .version }}
section_menu_id: setup
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Installation Guide

Voyager operator can be installed via a script or as a Helm chart.

<ul class="nav nav-tabs" id="installerTab" role="tablist">
  <li class="nav-item">
    <a class="nav-link active" id="helm3-tab" data-toggle="tab" href="#helm3" role="tab" aria-controls="helm3" aria-selected="true">Helm 3 (Recommended)</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="helm2-tab" data-toggle="tab" href="#helm2" role="tab" aria-controls="helm2" aria-selected="false">Helm 2</a>
  </li>
  <li class="nav-item">
    <a class="nav-link" id="script-tab" data-toggle="tab" href="#script" role="tab" aria-controls="script" aria-selected="false">YAML</a>
  </li>
</ul>
<div class="tab-content" id="installerTabContent">
  <div class="tab-pane fade show active" id="helm3" role="tabpanel" aria-labelledby="helm3-tab">

## Using Helm 3

Voyager can be installed via [Helm](https://helm.sh/) 3.x or later versions using the [chart](https://github.com/appscode/voyager/tree/{{< param "info.version" >}}/charts/voyager) from [AppsCode Charts Repository](https://github.com/appscode/charts). To install the chart with the release name `my-release`:

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm search repo appscode/voyager --version {{< param "info.version" >}}
NAME              CHART VERSION APP VERSION DESCRIPTION
appscode/voyager  {{< param "info.version" >}}    {{< param "info.version" >}}  Voyager by AppsCode - Secure HAProxy Ingress Controller...

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

$ helm install voyager-operator appscode/voyager --version {{< param "info.version" >}} \
  --namespace kube-system \
  --set cloudProvider=$provider
```

To see the detailed configuration options, visit [here](https://github.com/appscode/voyager/tree/{{< param "info.version" >}}/charts/voyager).

</div>
<div class="tab-pane fade" id="helm2" role="tabpanel" aria-labelledby="helm2-tab">

## Using Helm 2

Voyager can be installed via [Helm](https://helm.sh/) 2.9.x or later versions using the [chart](https://github.com/appscode/voyager/tree/{{< param "info.version" >}}/charts/voyager) from [AppsCode Charts Repository](https://github.com/appscode/charts). To install the chart with the release name `my-release`:

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm search appscode/voyager --version {{< param "info.version" >}}
NAME              CHART VERSION APP VERSION DESCRIPTION
appscode/voyager  {{< param "info.version" >}}    {{< param "info.version" >}}  Voyager by AppsCode - Secure HAProxy Ingress Controller...

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

$ helm install appscode/voyager --name voyager-operator --version {{< param "info.version" >}} \
  --namespace kube-system \
  --set cloudProvider=$provider
```

To see the detailed configuration options, visit [here](https://github.com/appscode/voyager/tree/{{< param "info.version" >}}/charts/voyager).

</div>
<div class="tab-pane fade" id="script" role="tabpanel" aria-labelledby="script-tab">

## Using YAML

If you prefer to not use Helm, you can generate YAMLs from Voyager operator chart and deploy using `kubectl`. Here we are going to show the prodecure using Helm 3.

```console
$ helm repo add appscode https://charts.appscode.com/stable/
$ helm repo update
$ helm search repo appscode/voyager --version {{< param "info.version" >}}
NAME              CHART VERSION APP VERSION DESCRIPTION
appscode/voyager  {{< param "info.version" >}}    {{< param "info.version" >}}  Voyager by AppsCode - Secure HAProxy Ingress Controller...

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

$ helm template voyager-operator appscode/voyager --version {{< param "info.version" >}} \
  --namespace kube-system \
  --no-hooks \
  --set cloudProvider=$provider | kubectl apply -f -
```

To see the detailed configuration options, visit [here](https://github.com/appscode/voyager/tree/{{< param "info.version" >}}/charts/voyager).

</div>
</div>

### Installing in GKE Cluster

If you are installing Voyager on a GKE cluster, you will need cluster admin permissions to install Voyager operator. Run the following command to grant admin permision to the cluster.

```console
$ kubectl create clusterrolebinding "cluster-admin-$(whoami)" \
  --clusterrole=cluster-admin \
  --user="$(gcloud config get-value core/account)"
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

Version = {{< param "info.version" >}}
VersionStrategy = tag
Os = alpine
Arch = amd64
CommitHash = ab0b38d8f5d5b4b4508768a594a9d98f2c76abd8
GitBranch = release-4.0
GitTag = {{< param "info.version" >}}
CommitTimestamp = 2017-10-08T12:45:26
```
