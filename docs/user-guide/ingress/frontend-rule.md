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
