---
menu:
  product_voyager_5.0.0-rc.10:
    name: Simple Fanout
    parent: http
    weight: 100
product_name: voyager
menu_name: product_voyager_5.0.0-rc.10
section_menu_id: guides
---

### Simple Fanout
As described previously, pods within kubernetes have ips only visible on the cluster network. So, we need
something at the edge accepting ingress traffic and proxy-ing it to right endpoints. This component
is usually a highly available loadbalancer(s). An Ingress allows you to keep number of loadbalancers
down to a minimum, for example, a setup can be like:


```
foo.bar.com -> load balancer -> / foo    s1:80
                                / bar    s2:80
```

would require an Ingress such as:
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
      - path: "/foo"
        backend:
          serviceName: s1
          servicePort: '80'
      - path: "/bar"
        backend:
          serviceName: s2
          servicePort: '80'
```
The Ingress controller will provision an implementation specific loadbalancer that satisfies the Ingress,
as long as the services (s1, s2) exist. When it has done so, you will see the address of the loadbalancer under
the Status of Ingress.

In Voyager, **the order of rules and paths is important** as Voyager will use them in the order provided by user, instead of automatically reordering them. So, to add a catch-all service for all other paths, you can add a `/` path to the end.
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
      - path: "/foo"
        backend:
          serviceName: s1
          servicePort: '80'
      - path: "/bar"
        backend:
          serviceName: s2
          servicePort: '80'
      - path: "/"
        backend:
          serviceName: catch-all
          servicePort: '80'
```


## Next Reading
- [Virtual Hosting](named-virtual-hosting.md)
- [URL and Header Rewriting](header-rewrite.md)
- [TCP Loadbalancing](../tcp/tcp.md)
- [TLS Termination](../tls/tls.md)
- [Configure Custom Timeouts for HAProxy](../configuration/configure-timeouts.md)
