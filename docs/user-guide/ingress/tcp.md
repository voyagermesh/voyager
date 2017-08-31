### TCP LoadBalancing
TCP load balancing is one of the core features of AppsCode Ingress. AppsCode Ingress can handle
TCP Load balancing with or without TLS. One AppsCode Ingress can also be used to load balance both
HTTP and TCP together.

One Simple TCP Rule Would be:
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'

```

For this configuration, the loadbalancer will listen to `9898` port for incoming connections, and will
pass any request coming to it to the desired backend.

> For one Ingress Type you cannot have multiple rules listening to same port, even if they do not have
same `host`.
For TCP rules host parameters do not have much effective value.

## Next Reading
- [TLS Termination](tls.md)
