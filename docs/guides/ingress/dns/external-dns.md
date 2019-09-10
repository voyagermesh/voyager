---
title: Configure External DNS for Kubernetes Ingress
menu:
  product_voyager_v11.0.0:
    identifier: external-dns-dns
    name: External DNS
    parent: dns-ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_v11.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Configuring external DNS servers

[external-dns](https://github.com/kubernetes-incubator/external-dns) project can be used to configure external DNS servers for Voyager managed ingresses.

## LoadBalancer Ingress

For a [LoadBalancer](/docs/concepts/ingress-types/loadbalancer.md) type Ingress, apply `"external-dns.alpha.kubernetes.io/hostname"` annotation on the **service** that exposes HAProxy pods. This service should have a name like `voyager-{ingress-name}` in the same namespace of the Ingress object. Since, Voyager uses its own CRD for Ingress, `external-dns` project must use the service to discover loadbalancer ip.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress-voyager
  namespace: vdimov-dev
  annotations:
     ingress.appscode.com/annotations-service: |
         {
           "external-dns.alpha.kubernetes.io/hostname" : "voyager.example.com,voyager-1.example.com,voyager-2.example.com"
         }
     ingress.appscode.com/replicas: '2'
spec:
  rules:
  - host: voyager.example.com
    http:
      paths:
       - backend:
          serviceName: web
          servicePort: '80'
```

## NodePort Ingress

Since [v0.5.3](https://github.com/kubernetes-incubator/external-dns/releases/tag/v0.5.3), `external-dns` supports [NodePort](/docs/concepts/ingress-types/nodeport.md) ingress.


## HostPort Ingress

[HostPort](/docs/concepts/ingress-types/hostport.md) type Ingress is [supported by external-dns](https://github.com/kubernetes-incubator/external-dns/blob/v0.5.2/docs/tutorials/hostport.md). Here, apply `"external-dns.alpha.kubernetes.io/hostname"` annotation on the HAProxy **services**.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress-voyager
  namespace: vdimov-dev
  annotations:
     ingress.appscode.com/type: HostPort
     ingress.appscode.com/annotations-service: |
         {
           "external-dns.alpha.kubernetes.io/hostname" : "voyager.example.com,voyager-1.example.com,voyager-2.example.com"
         }
     ingress.appscode.com/replicas: '2'
spec:
  rules:
  - host: voyager.example.com
    http:
      paths:
       - backend:
          serviceName: web
          servicePort: '80'
```

## Internal Ingress

[Internal](/docs/concepts/ingress-types/internal.md) ingress is not accessible from outside a cluster. Hence, there is nothing to configure in external DNS servers.
