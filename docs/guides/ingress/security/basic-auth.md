---
title: Basic Authentication | Kubernetes Ingress
menu:
  product_voyager_7.2.0:
    identifier: basic-auth-security
    name: Basic Auth
    parent: security-ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_7.2.0
section_menu_id: guides
aliases:
  - /products/voyager/7.2.0/guides/ingress/security/
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Basic Authentication

This example demonstrates how to configure
[Basic Authentication](https://tools.ietf.org/html/rfc2617) on
Voyager Ingress controller.


## Using Basic Authentication

Voyager Ingress read user and password from files stored on secrets, one user
and password per line. Secret name, realm and type are configured with annotations
in the ingress resource:

* `ingress.appscode.com/auth-type`: the only supported type is `basic`
* `ingress.appscode.com/auth-realm`: an optional string with authentication realm
* `ingress.appscode.com/auth-secret`: name of the secret

Each line of the `auth` file should have:

* user and insecure password separated with a pair of colons: `<username>::<plain-text-password>`; or
* user and an encrypted password separated with colons: `<username>:<encrypted-passwd>`

If passwords are provided in plain text, Voyager operator will encrypt them before rendering HAProxy configuration.
HAProxy evaluates encrypted passwords with [crypt](http://man7.org/linux/man-pages/man3/crypt.3.html) function. Use `mkpasswd` or
`makepasswd` to create it. `mkpasswd` can be found on Alpine Linux container.

### Configure

Create a secret to our users:

* `john` and password `admin` using insecure plain text password
* `jane` and password `guest` using encrypted password

```console
$ mkpasswd -m des ## a short, des encryption, syntax from Busybox on Alpine Linux
Password: (type 'guest' and press Enter)
E5BrlrQ5IXYK2

$ cat >auth <<EOF
john::admin
jane:E5BrlrQ5IXYK2
EOF

$ kubectl create secret generic mypasswd --from-file auth
$ rm -fv auth

# run test servers
$ kubectl run nginx --image=nginx
$ kubectl expose deployment nginx --name=web --port=80 --target-port=80
```

Create an Ingress with Basic Auth annotations

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/rewrite-target: /
    ingress.appscode.com/auth-type: basic
    ingress.appscode.com/auth-realm: My Server
    ingress.appscode.com/auth-secret: mypasswd
  name: basic-auth-ingress
  namespace: default
spec:
  rules:
  - http:
      paths:
      - path: /web
        backend:
          serviceName: web
          servicePort: 80
```

Test without user and password:

```console
$ curl -i ip:port
HTTP/1.0 401 Unauthorized
Cache-Control: no-cache
Connection: close
Content-Type: text/html
Authentication problem. Ignoring this.
WWW-Authenticate: Basic realm="My Server"

<html><body><h1>401 Unauthorized</h1>
You need a valid user and password to access this content.
</body></html>
```

Send a valid user:

```console
$ curl -i -u 'john:admin' ip:port
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

Using `jane:guest` user/passwd should have the same output.

## Using Basic Auth for backend service

Voyager Ingress can be configured to use Basic Auth per Backend service by applying the annotations to
kubernetes service.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: test-svc
  namespace: default
  annotations:
    ingress.appscode.com/auth-type: basic
    ingress.appscode.com/auth-realm: My Server
    ingress.appscode.com/auth-secret: mypasswd
spec:
  ports:
  - name: http-1
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: deployment
```

Create an Ingress with Basic Auth only on path `/auth`

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: hello-basic-auth
  namespace: default
spec:
  rules:
  - http:
      paths:
      - path: /no-auth
        backend:
          serviceName: test-server
          servicePort: 80
  - http:
      paths:
      - path: /auth
        backend:
          serviceName: test-svc
          servicePort: 80

```

Test without user and password:

```console
$ curl -i ip:port/auth
HTTP/1.0 401 Unauthorized
Cache-Control: no-cache
Connection: close
Content-Type: text/html
Authentication problem. Ignoring this.
WWW-Authenticate: Basic realm="My Server"

<html><body><h1>401 Unauthorized</h1>
You need a valid user and password to access this content.
</body></html>
```

Send a valid user:

```console
$ curl -i -u 'john:admin' ip:port/auth
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

No auth enabled Backend

```console
$ curl -i ip:port/no-auth
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

## Using Basic Auth In Frontend

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
      basic:
        secretName: mypasswd
        realm: My Server
  rules:
  - http:
      port: '80'
      paths:
      - path: /no-auth
        backend:
          serviceName: test-server
          servicePort: 80
  - http:
      port: '8080'
      paths:
      - path: /auth
        backend:
          serviceName: test-svc
          servicePort: 80

```

Test without user and password:

```console
$ curl -i ip:8080/auth
HTTP/1.0 401 Unauthorized
Cache-Control: no-cache
Connection: close
Content-Type: text/html
Authentication problem. Ignoring this.
WWW-Authenticate: Basic realm="My Server"

<html><body><h1>401 Unauthorized</h1>
You need a valid user and password to access this content.
</body></html>
```

Send a valid user:

```console
$ curl -i -u 'john:admin' ip:8080/auth
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

No auth enabled Backend
```console
$ curl -i ip:9090/no-auth
HTTP/1.1 200 OK
Date: Fri, 08 Sep 2017 09:31:43 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

## Acknowledgement
  - This document has been adapted from [kubernetes/ingress](https://github.com/kubernetes/ingress/tree/master/examples/auth/basic/haproxy) project.
