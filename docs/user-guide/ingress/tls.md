## TLS
You can secure an Ingress by specifying a secret containing TLS pem or By referring a `certificate.appscode.com` resource.
Referring `certificate.appscode.com` will try to manage an certificate resource and use that certificate to encrypt communication.
We will discuss those things later.
Currently the Ingress only supports a
single TLS port, **443 for HTTP Rules**, and **Any Port for TCP Rules** and **assumes TLS termination**.

### HTTP TLS
For HTTP, If the TLS configuration section in an Ingress specifies different hosts, they will be multiplexed
on the same port according to hostname specified through SNI TLS extension
(Ingress controller supports SNI). The TLS secret must contain pem file to use for TLS, with a
key name ending with `.pem`. eg:

```
apiVersion: v1
kind: Secret
metadata:
  name: testsecret
  namespace: default
data:
  tls.crt: base64 encoded cert
  tls.key: base64 encoded key
```

Referencing this secret in an Ingress will tell the Ingress controller to secure the channel from
client to the loadbalancer using TLS:
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
Adding a TCP TLS termination at AppsCode Ingress is slightly different than HTTP, as TCP do not have
SNI advantage. An TCP endpoint with TLS termination, will look like this in AppsCode Ingress:
```yaml
apiVersion: appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - tcp:
    - host: appscode.example.com
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
      secretName: testsecret

```
You need to set  the secretName field with the TCP rule to use a certificate.
