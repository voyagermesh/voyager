### Setup NodePort
Voyager 3.2+ supports defining NodePort.

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
      nodePort: '32666'
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  - host: appscode.example.com
    tcp:
      port: '4343'
      nodePort: '35666'
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'

```

For this configuration, the loadbalancer will listen to `8989` and `4343` port for incoming HTTP
connections and these port will map to specified nodeports, and will pass any request coming to it to the desired backend.

> For one Ingress Type you cannot have multiple rules listening to same port or same node port, even if they do not have
same `host`.
