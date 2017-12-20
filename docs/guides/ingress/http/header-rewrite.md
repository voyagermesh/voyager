---
menu:
  product_voyager_5.0.0-rc.8:
    name: Header Rewrite
    parent: ingress
    weight: 70
product_name: voyager
menu_name: product_voyager_5.0.0-rc.8
section_menu_id: guides
---


### Header and URL Rewriting
AppsCode Ingress support header and URL modification at the loadbalancer level. To ensure simplicity,
the header and rewrite rules follow the HAProxy syntax as it is.
To add some rewrite rules in a http rule, the syntax is:
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
          headerRule:
          - X-Forwarded-Host %[base]
          rewriteRule:
          - "^([^\\ :]*)\\ /(.*)$ \\1\\ /testings/\\2"
```
The rules specified in `headerRule` will be applicable to the request header before going to the backend.
those rules will be added in the request header if the header is already not present in the request header.
In the example `X-Forwarded-Host` header is added to the request if it is not already there, `%[base]` indicates
the base URL the load balancer received the requests.

The rules specified in `rewriteRule` are used to modify the request url including the host. Current example
will add an `/testings` prefix in every request URI before forwarding it to backend.

## Next Reading
- [TCP Loadbalancing](tcp.md)
- [TLS Termination](tls.md)
