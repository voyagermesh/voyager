## HSTS
HTTP Strict Transport Security (HSTS) is a web security policy mechanism which helps to protect
websites against protocol downgrade attacks and cookie hijacking. It allows web servers to
declare that web browsers (or other complying user agents) should only interact with it using secure
HTTPS connections, and never via the insecure HTTP protocol. HSTS is an IETF standards track protocol and is specified in RFC 6797.

The HSTS Policy is communicated by the server to the user agent via an HTTPS response header field named "Strict-Transport-Security".
HSTS Policy specifies a period of time during which the user agent should only access the server in a secure fashion.

voyager can insert HSTS header in http response if configured via ingress annotations.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: my-voyager
  namespace: default
  annotation:
    ingress.kubernetes.io/hsts: "true"
    ingress.kubernetes.io/hsts-preload: "true"
    ingress.kubernetes.io/hsts-include-subdomains: "true"
    ingress.kubernetes.io/hsts-max-age: 100
spec:
  tls:
    - secretName: test-secret
    hosts: foo.bar.com
  rules:
  - host: foo.bar.com
    http:
      paths:
      - path: /two
        backend:
          serviceName: server-2
          servicePort: 9090
```

Applying the annotation in ingress will have the following effects, will add the HSTS Header in the response.
```console
$ curl -v -X 'GET' -k 'http://foo.bar.com'
Strict-Transport-Security: max-age=100; includeSubDomains; preload
```
