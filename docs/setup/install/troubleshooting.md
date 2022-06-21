---
title: Troubleshooting Voyager Installation
description: Troubleshooting guide for Voyager installation
menu:
  docs_{{ .version }}:
    identifier: install-voyager-troubleshoot
    name: Troubleshooting
    parent: installation-guide
    weight: 40
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: setup
---

## Installing in GKE Cluster

If you are installing Voyager on a GKE cluster, you will need cluster admin permissions to install Voyager operator. Run the following command to grant admin permission to the cluster.

```bash
$ kubectl create clusterrolebinding "cluster-admin-$(whoami)" \
  --clusterrole=cluster-admin                                 \
  --user="$(gcloud config get-value core/account)"
```

In addition, if your GKE cluster is a [private cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters), you will need to either add an additional firewall rule that allows master nodes access port `8443/tcp` on worker nodes, or change the existing rule that allows access to ports `443/tcp` and `10250/tcp` to also allow access to port `8443/tcp`. The procedure to add or modify firewall rules is described in the official GKE documentation for private clusters mentioned before.

### Installing in kind

Voyager can be used in kind using `--provider=kind`. In kind, a `LoadBalancer` type ingress will only assigned a NodePort.

### Installing in Baremetal Cluster

Voyager works great in baremetal cluster. To install, set `--provider=baremetal`. In baremetal cluster, `LoadBalancer` type ingress in not supported. You can use [NodePort](/docs/concepts/ingress-types/nodeport.md), [HostPort](/docs/concepts/ingress-types/hostport.md) or [Internal](/docs/concepts/ingress-types/internal.md) ingress objects.

### Installing in Baremetal Cluster with MetalLB

Follow the instructions for installing on baremetal cluster but specify `metallb` as provider. Then install MetalLB following the instructions [here](https://metallb.universe.tf/installation/). Now, you can use `LoadBalancer` type ingress in baremetal clusters.

### Installing in DigitalOcean Cluster

To use `LoadBalancer` type ingress in [DigitalOcean](https://www.digitalocean.com/) cluster, install Kubernetes [cloud controller manager for DigitalOcean](https://github.com/digitalocean/digitalocean-cloud-controller-manager). Otherwise set cloud provider to `baremetal`.

### Installing in Linode Cluster

To use `LoadBalancer` type ingress in [Linode](https://www.linode.com/) cluster, install Kubernetes [cloud controller manager for Linode](https://github.com/pharmer/cloud-controller-manager). Otherwise set cloud provider to `baremetal`.

## Detect Voyager version

To detect Voyager version, exec into the operator pod and run `voyager version` command.

```bash
$ POD_NAMESPACE=voyager
$ POD_NAME=$(kubectl get pods -n $POD_NAMESPACE -l app.kubernetes.io/name=voyager -o jsonpath={.items[0].metadata.name})
$ kubectl exec $POD_NAME -c operator -n $POD_NAMESPACE -- /voyager version

Version = v13.0.0
VersionStrategy = tag
GitTag = v13.0.0
GitBranch = HEAD
CommitHash = 2c9d9b810413c620c45ea028b84262704ebcea54
CommitTimestamp = 2021-09-16T01:59:11
GoVersion = go1.17.1
Compiler = gcc
Platform = linux/amd64
```
