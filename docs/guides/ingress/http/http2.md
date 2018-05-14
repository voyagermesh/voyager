# Enable HTTP/2 on ingress resource

Voyager can enable HTTP/2 from version >=7.0.0-rc.1

To enable http2, you must first setup a [certificate](https://appscode.com/products/voyager/6.0.0/guides/certificate/) (Let's Encrypt), or use an existing one then create an ingress resource configuration like so:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: haproxy-ingress # name of the ingress
  namespace: default # namespace (optional)
  annotations:
    ingress.appscode.com/replicas: '2'
    # ... other annotations
spec:
  tls:
  - hosts:
    - host.example.com
    ref:
      kind: Certificate
      name: host
  rules:
  - host: host.example.com
    http:
      paths:
      - path: "/"
        backend:
          serviceName: host-service
          servicePort: '8000'
      alpn:
      - h2
      - http/1.1
      - http/1.0
```

The important bit here is `alpn`. In this example the ingress resource will first offer `HTTP/2`, then `HTTP/1.1` and finally `HTTP/1.0` to the connecting client. If it's not specified, it defaults to HTTP/1.1

The above settings are reflected in the generated haproxy `ConfigMap`:

```frontend http-0_0_0_0-443
    bind *:443  ssl no-sslv3 no-tlsv10 no-tls-tickets crt /etc/ssl/private/haproxy/tls/  alpn h2,http/1.1,http/1.0
```
