# Sticky Session
Voyager 3.2.+ supports two different configuration to allow [sticky connection](https://www.haproxy.com/blog/load-balancing-affinity-persistence-sticky-sessions-what-you-need-to-know/) backend
servers. By applying annotation to ingress resource we can configure all backend
in that ingress to allow sticky session. Or we can apply annotation to target service and configure
specific backends to allow sticky session.


### Sticky Ingress
Applying annotation `ingress.appscode.com/sticky-session` to ingress will configure all backend to
support sticky session.

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
For the above ingress all three backend connection will be sticky.

### Sticky Backends
Applying annotation `ingress.appscode.com/sticky-session` to service will only configure that specific
backend to use sticky connection.

As an example, The following Service and Ingress will only configure sticky connections to
s1 backends.

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
  - protocol: TCP
    port: 80
    targetPort: 9376
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
      - backend:
          serviceName: s1   # Only this is sticky
          servicePort: '80'
  - host: bar.foo.com
    http:
      paths:
      - backend:
          serviceName: s2   # No sticky
          servicePort: '80'
  - host: tcp.bar.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
```