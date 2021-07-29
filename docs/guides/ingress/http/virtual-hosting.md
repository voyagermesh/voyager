---
title: Virtual Hosting | Kubernetes Ingress
menu:
  docs_{{ .version }}:
    identifier: virtual-hosting-http
    name: Virtual Hosting
    parent: http-ingress
    weight: 15
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Virtual Hosting

## Hostname based Routing

Name-based virtual hosts use multiple host names for the same IP address.

```
foo.bar.com --|               |-> foo.bar.com s1:80
              | load balancer |
bar.foo.com --|               |-> bar.foo.com s2:80
```
The following Ingress tells the backing loadbalancer to route requests based on the [Host header](https://tools.ietf.org/html/rfc7230#section-5.4).

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          service:
            name: s1
            port:
              number: 80
  - host: bar.foo.com
    http:
      paths:
      - backend:
          service:
            name: s2
            port:
              number: 80
```

> AppsCode Ingress also support **wildcard** Name based virtual hosting.
If the `host` field is set to `*.bar.com`, Ingress will forward traffic for any subdomain of `bar.com`.
so `foo.bar.com` or `test.bar.com` will forward traffic to the desired backends.

## Cross Namespace traffic routing
If your ingress in namespace `foo` and your application is in namespace `bar` you can still forward traffic.

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: foo
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          service:
            name: s1.bar # serviceName.Namespace
            port:
              number: 80
```

## Path based Routing

A setup can be like:

```
foo.bar.com -> load balancer -> / foo    s1:80
                                / bar    s2:80
```

would require an Ingress such as:

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - path: "/foo"
        backend:
          service:
            name: s1
            port:
              number: 80
      - path: "/bar"
        backend:
          service:
            name: s2
            port:
              number: 80
```

The Ingress controller will provision an implementation specific loadbalancer that satisfies the Ingress,
as long as the services (s1, s2) exist.
