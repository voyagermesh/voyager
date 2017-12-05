---
menu:
  product_voyager_5.0.0-rc.6:
    name: Backend Rule
    parent: ingress
    weight: 20
product_name: voyager
menu_name: product_voyager_5.0.0-rc.6
section_menu_id: user-guide
---

### BackendRule
Voyager supports full spectrum of HAProxy backend rules via `backendRule`. Read [more](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html)
about HAProxy backend rules.

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
      - path: '/test'
        backend:
          serviceName: test-service
          servicePort: '80'
          backendRule:
          - 'acl add_url capture.req.uri -m beg /test-second'
          - 'http-response set-header X-Added-From-Proxy added-from-proxy if add_url'
```

This example will apply an acl to the server backend, and a extra header from Loadbalancer if request uri
starts with `/test-second`.