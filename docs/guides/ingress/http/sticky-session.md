---
title: Sticky Session | Kubernetes Ingress
menu:
  product_voyager_6.0.0-rc.0:
    identifier: sticky-http
    name: Sticky Session
    parent: http-ingress
    weight: 55
product_name: voyager
menu_name: product_voyager_6.0.0-rc.0
section_menu_id: guides
---

# Sticky Session

Voyager 3.2.0+ can configure [sticky connections](https://www.haproxy.com/blog/load-balancing-affinity-persistence-sticky-sessions-what-you-need-to-know/) in 2 modes. By applying annotation to an Ingress resource, you can configure all backends in that ingress to use sticky session. Or you can apply annotation to a service and configure
backends using that service to use sticky session.

`ingress.appscode.com/sticky-session` annotations is deprecated in voyager 4.0.0+ and removed in 6.0.0+. Use `ingress.appscode.com/affinity` instead.

## Sticky Ingress

Applying annotation `ingress.appscode.com/affinity` to Ingress will configure all backends to support sticky session.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotation:
    ingress.appscode.com/affinity: 'cookie'
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          serviceName: s1
          servicePort: '80'
  - host: bar.foo.com
    http:
      paths:
      - backend:
          serviceName: s2
          servicePort: '80'
  - host: tcp.bar.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
```

For the above ingress, all three backend connections will be sticky.

## Sticky Service

Applying annotation `ingress.appscode.com/affinity` to a service will configures any backend
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
apiVersion: voyager.appscode.com/v1beta1
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
          serviceName: s1   # Sticky, since service s1 is annotated
          servicePort: '5050'
      - path: /
        backend:
          serviceName: s1   # Sticky, since service s1 is annotated
          servicePort: '80'
  - host: bar.foo.com
    http:
      paths:
      - backend:
          serviceName: s2   # Not sticky
          servicePort: '80'
  - host: tcp.bar.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service # Not sticky
        servicePort: '50077'
```

## Other Annotations

- `ingress.appscode.com/session-cookie-name`: When affinity is set to `cookie`, the name of the cookie to use.
- `ingress.appscode.com/session-cookie-hash`: When affinity is set to `cookie`, the hash algorithm used: md5, sha, index.