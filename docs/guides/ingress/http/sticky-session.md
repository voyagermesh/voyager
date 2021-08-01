---
title: Sticky Session | Kubernetes Ingress
menu:
  docs_{{ .version }}:
    identifier: sticky-http
    name: Sticky Session
    parent: http-ingress
    weight: 55
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Sticky Session

Voyager 3.2.0+ can configure [sticky connections](https://www.haproxy.com/blog/load-balancing-affinity-persistence-sticky-sessions-what-you-need-to-know/) in 2 modes. By applying annotation to an Ingress resource, you can configure all backends in that ingress to use sticky session. Or you can apply annotation to a service and configure
backends using that service to use sticky session.

`ingress.appscode.com/sticky-session` annotations is deprecated in voyager 4.0.0+ and removed in 8.0.1+. Use `ingress.appscode.com/affinity` instead.

## Sticky Ingress

Applying annotation `ingress.appscode.com/affinity` to Ingress will configure all backends to support sticky session.

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/affinity: 'cookie'
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
  - host: tcp.bar.com
    tcp:
      port: 9898
      backend:
        service:
          name: tcp-service
          port:
            number: 50077
```

For the above ingress, all three backend connections will be sticky.

## Sticky Service

Applying annotation `ingress.appscode.com/affinity` to a service configures any backend
that uses that service to use sticky connection. As an example, the following Ingress will only
configure sticky connections for backends that use `s1` Service.

```yaml
kind: Service
apiVersion: v1
metadata:
  name: s1
  namespace: default
  annotations:
    ingress.appscode.com/affinity: 'cookie'
spec:
  selector:
    app: app
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 9376
  - name: ops
    protocol: TCP
    port: 5050
    targetPort: 5089
```

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
      - path: /admin
        backend:
          service:
            name: s1   # Sticky, since service s1 is annotated
            port:
              number: 5050
      - path: /
        backend:
          service:
            name: s1   # Sticky, since service s1 is annotated
            port:
              number: 80
  - host: bar.foo.com
    http:
      paths:
      - backend:
          service:
            name: s2   # Not sticky
            port:
              number: 80
  - host: tcp.bar.com
    tcp:
      port: 9898
      backend:
        service:
          name: tcp-service # Not sticky
          port:
            number: 50077
```

## Other Annotations

- `ingress.appscode.com/session-cookie-name`: When affinity is set to `cookie`, the name of the cookie to use.
- `ingress.appscode.com/session-cookie-hash`: When affinity is set to `cookie`, the hash algorithm used: md5, sha, index.
