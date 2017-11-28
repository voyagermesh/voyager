---
menu:
  product_voyager_5.0.0-rc.3:
    name: Named Virtual Hosting
    parent: ingress
    weight: 85
product_name: voyager
left_menu: product_voyager_5.0.0-rc.3
section_menu_id: user-guide
---


### Name based virtual hosting
Name-based virtual hosts use multiple host names for the same IP address.

```
foo.bar.com --|               |-> foo.bar.com s1:80
              | load balancer |
bar.foo.com --|               |-> bar.foo.com s2:80
```
The following Ingress tells the backing loadbalancer to route requests based on the [Host header](https://tools.ietf.org/html/rfc7230#section-5.4).

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
          serviceName: s1
          servicePort: '80'
  - host: bar.foo.com
    http:
      paths:
      - backend:
          serviceName: s2
          servicePort: '80'
```

> AppsCode Ingress also support **wildcard** Name based virtual hosting.
If the `host` field is set to `*.bar.com`, Ingress will forward traffic for any subdomain of `bar.com`.
so `foo.bar.com` or `test.bar.com` will forward traffic to the desired backends.

### Cross Namespace traffic routing
If your ingress in namespace `foo` and your application is in namespace `bar` you can still forward traffic.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: foo
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          serviceName: s1.bar # serviceName.Namespace
          servicePort: '80'
```


## Next Reading
- [URL and Header Rewriting](header-rewrite.md)
- [TCP Loadbalancing](tcp.md)
- [TLS Termination](tls.md)
