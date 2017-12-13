---
menu:
  product_voyager_5.0.0-rc.7:
    name: Frontend Rule
    parent: ingress
    weight: 65
product_name: voyager
menu_name: product_voyager_5.0.0-rc.7
section_menu_id: tasks
---


## Frontend Rules
Frontend rules specify a set of rules that are applied to HAProxy frontend configuration.
The set of keywords are from here https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#4.1.
Only frontend sections can be applied here. **It is up to user to provide valid sets of rules**.
This allows acls or other options in frontend sections in HAProxy config.
Frontend rules will be mapped to `spec.rules` according to HAProxy port.


```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  frontendRules:
  - port: 80  # Applies all the rule in frontend section for port 80
    rules:
    - timeout client 5s   # Set the maximum inactivity time on the client side.
  - port: 9898 # Applies all the rule in frontend section for port 9898
    rules:
    - timeout client 500s # Set the maximum inactivity time on the client side.
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

This example ingress shows how to configure frontend rules in ingress resource. All the frontend rules for port 80
will be applied to all the backends which listens to port 80.


## Example: Whitelist IP Addresses using Frontend Rules
This example demonstrates How to whitelist some IP addresses for a backend using frontend rule.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotation:
    ingress.appscode.com/keep-source-ip: "true"
spec:
  frontendRules:
  - port: 80
    rules:
    # you can use IP addresses but also networks in the src acl. Both 192.168.20.0/24 and 192.168.10.3 work.
    - acl network_allowed src 128.196.0.5 128.196.0.5
    - block if !network_allowed
  - port: 9898
    rules:
    - acl network_allowed src 20.30.40.50 8.9.9.0/27
    - tcp-request connection reject if !network_allowed
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          serviceName: s1
          servicePort: '80'
  - host: tcp.bar.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
```

## FAQ

### Why does not IP whitelisting work in LoadBalancer type Ingress in AWS?

From [HAProxy official documentation](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.1-accept-proxy):
 ```
The PROXY protocol dictates the layer 3/4 addresses of the incoming connection
to be used everywhere an address is used, with the only exception of
"tcp-request connection" rules which will only see the real connection address.
```

The issue is that `keep-source-ip: true` annotation will enable `accept-proxy` option in HAProxy. But HAProxy does not use the IP received via PROXY protocol with `tcp-request connection reject`. Instead HAProxy uses the real IP it detected (which is the IP address of ELB in this case). This is actually an important security feature. Otherwise, any one can open a TCP connection and spoof their IP using the PROXY protocol header and by pass the whitelist. This works with HTTP on the backend rules, because in HTTP mode, HAPproxy checks the header and by using `accept-proxy`, we have told HAProxy to trust the header in PROXY protocol. So, for TCP connections that are behind ELB, you need to reject connection at ELB layer using `spec.loadBalancerSourceRanges` in Ingress. If you were running a bare-metal cluster, where traffic directly hits HAProxy, `tcp-request connection reject` will behave as expected.
