---
title: TLS Authentication | Kubernetes Ingress
menu:
  product_voyager_5.0.0-rc.11:
    identifier: tls-auth-security
    name: TLS Auth
    parent: security-ingress
    weight: 15
product_name: voyager
menu_name: product_voyager_5.0.0-rc.11
section_menu_id: guides
---

# TLS Authentication

This example demonstrates how to configure [TLS Authentication](https://tools.ietf.org/html/rfc2617) on Voyager Ingress controller.

- [Using tls auth in Ingress](#using-tls-authentication)
- [Using tls auth in Frontend](#using-tls-auth-in-frontend)

Before diving into the deep learn about TLS Auth with HAproxy.
- [SSL Client certificate management at application level](https://www.haproxy.com/blog/ssl-client-certificate-management-at-application-level/)
- [Clinet side ssl certificates](https://raymii.org/s/tutorials/haproxy_client_side_ssl_certificates.html)

## Using TLS Authentication

Voyager Ingress read ca certificates from files stored on secrets with `ca.crt` key.

* `ingress.kubernetes.io/auth-tls-secret`: Name of secret for TLS client certification validation.
* `ingress.kubernetes.io/auth-tls-error-page`: The page that user should be redirected in case of Auth error
* `ingress.kubernetes.io/auth-tls-verify-client`: Enables verification option of client certificates.

### Configure

Create tls secret for enable ssl termination:

```console
$ kubectl create secret tls server --cert=/path/to/cert/file --key=/path/to/key/file
```

Create ca cert secret:

```console
$ kubectl create secret generic ca --from-file=/path/to/ca.crt
```

Create an Ingress with TLS Auth annotations

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.kubernetes.io/auth-tls-secret: ca
    ingress.kubernetes.io/auth-tls-verify-client: required
    ingress.kubernetes.io/auth-tls-error-page: "https://auth.example.com/errors.html"
  name: hello-tls-auth
  namespace: default
spec:
  tls:
  - ref:
      kind: Secret
      name: server
    hosts:
    - auth.example.com
  rules:
  - host: auth.example.com
    http:
      paths:
      - path: /testpath
        backend:
          serviceName: test-server
          servicePort: 80
```

Test without certificates:

```console
$ curl -i -vvv 'https://auth.example.com'
* Hostname was NOT found in DNS cache
*   Trying 192.168.99.100...
* Connected to http.appscode.test (192.168.99.100) port 443 (#0)
* successfully set certificate verify locations:
*   CAfile: none
  CApath: /etc/ssl/certs
* SSLv3, TLS handshake, Client hello (1):
* SSLv3, TLS handshake, Server hello (2):
* SSLv3, TLS handshake, CERT (11):
* SSLv3, TLS handshake, Server key exchange (12):
* SSLv3, TLS handshake, Request CERT (13):
* SSLv3, TLS handshake, Server finished (14):
* SSLv3, TLS handshake, CERT (11):
* SSLv3, TLS handshake, Client key exchange (16):
* SSLv3, TLS change cipher, Client hello (1):
* SSLv3, TLS handshake, Finished (20):
* SSLv3, TLS alert, Server hello (2):
* error:14094410:SSL routines:SSL3_READ_BYTES:sslv3 alert handshake failure
* Closing connection 0
curl: (35) error:14094410:SSL routines:SSL3_READ_BYTES:sslv3 alert handshake failure
```

Send a valid clinet certificate:

```console
$ curl -v -s --key client.key --cert client.crt https://auth.example.com
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

Send a invalid clinet certificate, that will redirect to error page if provided:

```console
$ curl -v -s --key invalidclient.key --cert invalidclient.crt https://auth.example.com
HTTP/1.1 302
Location: https://auth.example.com/errors.html
```

## Using TLS Auth In Frontend
Basic Auth can also be configured per frontend in voyager ingress via FrontendRules.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: hello-basic-auth
  namespace: default
spec:
  frontendRules:
  - port: '8080'
    auth:
      tls:
        secretName: server
        verifyClient: required
        errorPage: "https://auth.example.com/error.html"
        headers:
          X-SSL-Client-CN: "%{+Q}[ssl_c_s_dn(cn)]"  # Add headers to Request based on SSL verification
          X-SSL:           "%[ssl_fc]",
  tls:
  - ref:
      kind: Secret
      name: server
    hosts:
    - auth.example.com
  rules:
  - host: auth.example.com
    http:
      paths:
      - path: /no-auth
        backend:
          serviceName: test-server
          servicePort: 80
  - host: auth.example.com
    http:
      port: '8080'
      paths:
      - path: /auth
        backend:
          serviceName: test-svc
          servicePort: 80

```

Request in non tls port:

```console
$ curl -v -s https://auth.example.com
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

Test without certificates:

```console
$ curl -i -vvv 'https://auth.example.com:8080'
* Hostname was NOT found in DNS cache
*   Trying 192.168.99.100...
* Connected to http.appscode.test (192.168.99.100) port 443 (#0)
* successfully set certificate verify locations:
*   CAfile: none
  CApath: /etc/ssl/certs
* SSLv3, TLS handshake, Client hello (1):
* SSLv3, TLS handshake, Server hello (2):
* SSLv3, TLS handshake, CERT (11):
* SSLv3, TLS handshake, Server key exchange (12):
* SSLv3, TLS handshake, Request CERT (13):
* SSLv3, TLS handshake, Server finished (14):
* SSLv3, TLS handshake, CERT (11):
* SSLv3, TLS handshake, Client key exchange (16):
* SSLv3, TLS change cipher, Client hello (1):
* SSLv3, TLS handshake, Finished (20):
* SSLv3, TLS alert, Server hello (2):
* error:14094410:SSL routines:SSL3_READ_BYTES:sslv3 alert handshake failure
* Closing connection 0
curl: (35) error:14094410:SSL routines:SSL3_READ_BYTES:sslv3 alert handshake failure
```

Send a valid clinet certificate:

```console
$ curl -v -s --key client.key --cert client.crt https://auth.example.com:8080
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```
backend server will receive Headers `X-SSL` and `X-SSL-Client-CN`.

Send a invalid clinet certificate, that will redirect to error page if provided:

```console
$ curl -v -s --key invalidclient.key --cert invalidclient.crt https://auth.example.com:8080
HTTP/1.1 302
Location: https://auth.example.com/errors.html
```
