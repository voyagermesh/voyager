---
title: Header and URL Rewriting | Voayger
menu:
  product_voyager_7.1.0:
    identifier: rewrite-http
    name: Rewrite Support
    parent: http-ingress
    weight: 25
product_name: voyager
menu_name: product_voyager_7.1.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Header and URL Rewriting

AppsCode Ingress support header and URL modification at the loadbalancer level. To ensure simplicity,
the header and rewrite rules follow the HAProxy syntax as it is. To add some rewrite rules for a HTTP path, follow the example below:

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
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
          headerRules:
          - X-Forwarded-Host %[base]
          rewriteRules:
          - "^([^\\ :]*)\\ /(.*)$ \\1\\ /testings/\\2"
```
The rules specified in `headerRules` will be applicable to the request header before going to the backend.
those rules will be added in the request header if the header is already not present in the request header.
In the example `X-Forwarded-Host` header is added to the request if it is not already there, `%[base]` indicates
the base URL the load balancer received the requests.

The rules specified in `rewriteRules` are used to modify the request url including the host. Current example
will add an `/testings` prefix in every request URI before forwarding it to backend.
