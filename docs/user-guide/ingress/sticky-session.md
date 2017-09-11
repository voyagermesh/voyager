# Sticky Session
Voyager 3.2.0+ can configure [sticky connections](https://www.haproxy.com/blog/load-balancing-affinity-persistence-sticky-sessions-what-you-need-to-know/) in 2 modes. By applying annotation to an Ingress resource, you can configure all backends in that ingress to use sticky session. Or you can apply annotation to a service and configure
backends using that service to use sticky session.


### Sticky Ingress
Applying annotation `ingress.appscode.com/sticky-session` to Ingress will configure all backends to
support sticky session. This mode was supported in Voyager versions prior to release 3.2.0 .

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotation:
    ingress.appscode.com/sticky-session: 'true'
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


### Sticky Service
Applying annotation `ingress.appscode.com/sticky-session` to a service will configures any backend that uses that service to use sticky connection. As an example, the following Ingress will only configure sticky connections for backends that use `s1` Service.

```yaml
kind: Service
apiVersion: v1
metadata:
  name: s1
  namespace: default
  annotations:
    ingress.appscode.com/sticky-session: 'true'
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