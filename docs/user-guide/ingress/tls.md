---
menu:
  product_voyager_5.0.0-rc.5:
    name: TLS
    parent: ingress
    weight: 135
product_name: voyager
left_menu: product_voyager_5.0.0-rc.5
section_menu_id: user-guide
---


## TLS
You can secure an Ingress by specifying a secret containing TLS pem or by referring a `certificate.voyager.appscode.com` resource.
`certificate.voyager.appscode.com` can manage an certificate resource and use that certificate to encrypt communication.

### HTTP TLS
For HTTP, If the `spec.TLS` section in an Ingress specifies different hosts, they will be multiplexed
on the same port according to hostname specified through SNI TLS extension (Voyager supports SNI). The provided Secret must have the PEM formatted certificate under `tls.crt` key and the PEM formatted private key under `tls.key` key. For example,

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: testsecret
  namespace: default
data:
  tls.crt: base64 encoded cert
  tls.key: base64 encoded key
```

Referencing this secret in an Ingress will tell the Voyager to secure the channel from client to the loadbalancer using TLS:
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  tls:
  - secretName: testsecret
    hosts:
    - one.example.com
  rules:
  - host: one.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
```
This Ingress will open an `https` listener to secure the channel from the client to the loadbalancer,
terminate TLS at load balancer with the secret retried via SNI and forward unencrypted traffic to the
`test-service`.

### TCP TLS
Adding a TCP TLS termination at Voyager Ingress is slightly different than HTTP, as TCP mode does not have
SNI support. A TCP endpoint with TLS termination, will look like this in Voyager Ingress:
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  tls:
    - secretName: testsecret
      hosts:
      - appscode.example.com
  rules:
  - host: appscode.example.com
    tcp:
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'

```
You need to set  the secretName field with the TCP rule to use a certificate.

### Serve both TLS and non-TLS under same host
Voyager Ingress can support for TLS and non-TLS traffic for same host in both HTTP and TCP mode. To do that you need to specify `noTLS: true` for the corresponding rule. Here is an example:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  tls:
  - secretName: onecert
    hosts:
    - one.example.com
  rules:
  - host: one.example.com
    http:
      paths:
      - backend:
          serviceName: test-server
          servicePort: '80'
  - host: one.example.com
    http:
      noTLS: true
      paths:
      - backend:
          serviceName: test-server
          servicePort: '80'
  - host: one.example.com
    tcp:
      port: '7878'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
  - host: one.example.com
    tcp:
      port: '7800'
      noTLS: true
      backend:
        serviceName: tcp-service
        servicePort: '50077'
```

For this Ingress, HAProxy will open up 3 separate ports:

- port 443: This is used by `spec.rules[0]`. Passes traffic to pods behind test-server:80. Uses TLS, since `spec.TLS` has a matching host.

- port 80: This is used by `spec.rules[1]`. Passes traffic to pods behind test-server:80. __Uses no TLS__, even though `spec.TLS` has a matching host. This is because `http.noTLS` is set to true for this rule.

- port 7878: This is used by `spec.rules[2]`. Passes traffic to pods behind tcp-service:50077. Uses TLS, since `spec.TLS` has a matching host.

- port 7880: This is used by `spec.rules[3]`. Passes traffic to pods behind tcp-service:50077. __Uses no TLS__, even though `spec.TLS` has a matching host. This is because `tcp.noTLS` is set to true for this rule.
