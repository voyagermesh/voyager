---
menu:
  product_voyager_5.0.0-rc.6:
    name: Basic Auth
    parent: ingress
    weight: 30
product_name: voyager
menu_name: product_voyager_5.0.0-rc.6
section_menu_id: tutorials
---


# Basic Authentication

This example demonstrates how to configure
[Basic Authentication](https://tools.ietf.org/html/rfc2617) on
Voyager Ingress controller.

## Using Basic Authentication

Voyager Ingress read user and password from files stored on secrets, one user
and password per line. Secret name, realm and type are configured with annotations
in the ingress resource:

* `ingress.kubernetes.io/auth-type`: the only supported type is `basic`
* `ingress.kubernetes.io/auth-realm`: an optional string with authentication realm
* `ingress.kubernetes.io/auth-secret`: name of the secret

Each line of the `auth` file should have:

* user and insecure password separated with a pair of colons: `<username>::<plain-text-password>`; or
* user and an encrypted password separated with colons: `<username>:<encrypted-passwd>`

If passwords are provided in plain text, Voyager operator will encrypt them before rendering HAProxy configuration.
HAProxy evaluates encrypted passwords with [crypt](http://man7.org/linux/man-pages/man3/crypt.3.html) function. Use `mkpasswd` or
`makepasswd` to create it. `mkpasswd` can be found on Alpine Linux container.

## Configure

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
```

Create an Ingress with Basic Auth annotations
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.kubernetes.io/auth-type: basic
    ingress.kubernetes.io/auth-realm: My Server
    ingress.kubernetes.io/auth-secret: mypasswd
  name: hello-basic-auth
  namespace: default
spec:
  rules:
  - http:
      paths:
      - path: /testpath
        backend:
          serviceName: test-service
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
WWW-Authenticate: Basic realm="Realm returned"

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

## Acknowledgement
  - This document has been adapted from [kubernetes/ingress](https://github.com/kubernetes/ingress/tree/master/examples/auth/basic/haproxy) project.
