### Custom HTTP Port
Voyager 3.2+ supports opening http port in any non custom port.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      port: '8989'
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  - host: appscode.example.com
      http:
        port: '4343'
        paths:
        - backend:
            serviceName: test-service
            servicePort: '80'

```

For this configuration, the loadbalancer will listen to `8989` and `4343` port for incoming HTTP connections, and will
pass any request coming to it to the desired backend.

> For one Ingress Type you cannot have multiple rules listening to same port, even if they do not have
same `host`.
