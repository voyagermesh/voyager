---
title: Overview | Voyager
menu:
  docs_{{ .version }}:
    identifier: overview-concepts
    name: Overview
    parent: concepts
    weight: 10
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: concepts
---

# Voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.


## Ingress
Voyager provides L7 and L4 loadbalancing using Kubernetes standard [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting among other features. The following diagram shows how voyager operator works. Voyager also provides a custom [Ingress](/docs/guides/ingress) resource under `voyager.appscode.com` api group that extends the functionality of standard Ingress in a Kubernetes native way.

![voyager-ingress](/docs/images/ingress/voyager-ingress.png)

The above diagram shows how the Voyager operator works. When Voyager is [installed](/docs/setup/install.md) in a Kubernetes cluster, a pod named `voyager-operator-***` starts running in `voyager` namespace by default. This operator pod watches for Kubernetes Ingress resources and Voyager's own Ingress CRD. When an Ingress object is created, Voyager operator creates 3 Kubernetes resources in the same namespace of the Ingress:

- a Configmap named `voyager-${ingress-name}`: This contains the auto generated HAProxy configuration under `haproxy.cfg` key.

- a Deployment named `voyager-${ingress-name}`: This runs HAProxy pods that mounts the above configmap. Each pod has one container for HAProxy. This container also includes some additional binary to reload HAProxy when the respective configmap updates. This also includes logic for mounting and updating SSL secrets referenced in the corresponding Ingress resource. HAProxy pods can also contain a side-car container for exporting Prometheus ready metrics, if [enabled](/docs/guides/ingress/monitoring/haproxy-stats.md).

- a Service named  `voyager-${ingress-name}`: This Kubernetes Service exposes the above HAProxy pods to the internet. The type of Service can be configured by user via `ingress.appscode.com/type` annotation on the Ingress.

- a Service named  `voyager-${ingress-name}-stats`: This Kubernetes Service is used to expose Prometheus ready metrics for HAProxy pods. This service is always of type `ClusterIP` and only created if stats are [enabled](/docs/guides/ingress/monitoring/haproxy-stats.md).

## Certificate

Voyager can automagically provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](/docs/guides/certificate) resource.

- Provision free TLS certificates from Let's Encrypt,
- Manage issued certificates using a Kubernetes Third Party Resource,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.


## FAQ

**How do I run Voyager with other Ingress controllers in the same cluster?**

Yes, Voyager can be used to manager Ingress objects alongside with other ingress controller. Voyager comes with its own CRD called `Ingress` under api version `voyager.appscode.com/v1` . This CRD is not recognized by other ingress controllers that works with the Kubernetes official Ingress object under `networking.k8s.io/v1` api version.

By default, Voyager will also manage Kubernetes Ingress objects under `networking.k8s.io/v1` api version. Voyager can be configured to only handle default Kubernetes Ingress objects with ingress.class `voyager` . To do that, pass the flag `--ingress-class=voyager` in operator pod. After that 

```yaml
  annotations:
    kubernetes.io/ingress.class=voyager
```

**Why does Voyager creates separate HAProxy pods for each Ingress?**

Various Ingress controller behave differently regarding whether separate Ingress object will result in separate LoadBalancer.

- nginx: Nginx controller seem to combine Ingress objects into one and expose via same Nginx instance.
- gce: A separate GCE L7 is created for each Ingress object.
- Voyager: A separate HAProxy deployment is created for each Ingress.

There is not a clear indication what is the intended behavior from https://kubernetes.io/docs/concepts/services-networking/ingress/ . Reasons why we think one LB (GCE L7 / nginx / haproxy) per Ingress is a better choice:

1. This gives users choice/control whether they want to serve all service using the same IP or use separate IP. In voyager, to serve using same LB, users need to put them in the same Ingress.

2. I think ux is clearer with this mode. If user don't have access to other namespaces, they may not know what else is going on.

3. Voyager supports TCP. If 2 separate backend wants to use the same port, they need separate LB IP. Also, since all TCP services served via same LB is in the same YAML, it makes it easy to see what other ports are in use.

4. In Voyager, the order of paths for same host is important. We maintain this order in generated HAProxy config. If the Ingresses are auto merged, user loses this control. This might be ok, since paths are matched as the url prefix. But will not work, if other options are used.

5. Users can spread services across LB pods. If used with HPA, users can only scale up HAProxy deployments that they want.

6. We do soft reload HAProxy. This gives better isolation.



