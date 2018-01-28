---
title: Overview | Voyager
menu:
  product_voyager_6.0.0-alpha.0:
    identifier: overview-concepts
    name: Overview
    parent: concepts
    weight: 10
product_name: voyager
menu_name: product_voyager_6.0.0-alpha.0
section_menu_id: concepts
---

# Voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.


## Ingress
Voyager provides L7 and L4 loadbalancing using Kubernetes standard [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting among other features. The following diagram shows how voyager operator works. Voyager also provides a custom [Ingress](/docs/guides/ingress) resource under `voyager.appscode.com` api group that extends the functionality of standard Ingress in a Kubernetes native way.

![voyager-ingress](/docs/images/ingress/voyager-ingress.png)

The above diagram shows how the Voyager operator works. When Voyager is [installed](/docs/setup/install.md) in a Kubernetes cluster, a pod named `voyager-operator-***` starts running in `kube-system` namespace by default. This operator pod watches for Kubernetes Ingress resources and Voyager's own Ingress CRD. When an Ingress object is created, Voyager operator creates 3 Kubernetes resources in the same namespace of the Ingress:

- a Configmap named `voyager-${ingress-name}`: This contains the auo generated HAProxy configuration under `haproxy.cfg` key.

- a Deployment named `voyager-${ingress-name}`: This runs HAProxy pods that mounts the above configmap. Each pod has one container for HAProxy. This container also includes some additional binary to reload HAProxy when the respective configmap updates. This also includes logic for mounting and updating SSL secrets referenced in the corresponding Ingress resource. HAProxy pods can also contain a side-car container for exporting Prometheus ready metrics, if [enabled](/docs/guides/ingress/monitoring/stats.md).

- a Service named  `voyager-${ingress-name}`: This Kubernetes Service exposes the above HAProxy pods to the internet. The type of Service can be configured by user via `ingress.appscode.com/type` annotation on the Ingress.

- a Service named  `voyager-${ingress-name}-stats`: This Kubernetes Service is used to expose Prometheus ready metrics for HAProxy pods. This service is always of type `ClusterIP` and only created if stats are [enabled](/docs/guides/ingress/monitoring/stats.md).

## Certificate

Voyager can automagically provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](/docs/guides/certificate) resource.

- Provision free TLS certificates from Let's Encrypt,
- Manage issued certificates using a Kubernetes Third Party Resource,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.
