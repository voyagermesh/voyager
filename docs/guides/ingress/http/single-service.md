---
title: Exposing Service | Kubernetes Ingress
menu:
  product_voyager_6.0.0:
    identifier: single-svc-http
    name: Single Service
    parent: http-ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_6.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Exposing Service via Ingress

There are existing Kubernetes concepts which allows you to expose a single service. However, you can do so
through an AppsCode Ingress as well, simply by specifying a default backend with no rules.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  backend:
    serviceName: test-service
    servicePort: '80'
```

This will create a load balancer forwarding all traffic to `test-server` service, unconditionally. The
loadbalancer ip can be found inside `Status` Field of the loadbalancer described response. **If there are other
rules defined in Ingress then the loadbalancer will forward traffic to the `test-server` when no other `rule` is
matched.**

**As Example:**

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  backend:
    serviceName: test-service
    servicePort: '80'
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'

```
**Default Backend**: An Ingress with no rules, like the one shown in the previous section, sends all
traffic to a single default backend. You can use the same technique to tell a loadbalancer
where to find your website’s 404 page, by specifying a set of rules and a default backend.
Traffic is routed to your default backend if none of the Hosts in your Ingress matches the Host in
request header, and/or none of the paths match url of request.

This Ingress will forward traffic to `test-service` if request comes from the host `appscode.example.com` only.
Other requests will be forwarded to default backend.

Default Backend also supports `headerRules` and `rewriteRules`.
