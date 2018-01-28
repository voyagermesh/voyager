---
title: CORS | Kubernetes Ingress
menu:
  product_voyager_6.0.0-alpha.0:
    identifier: cors-http
    name: CORS
    parent: http-ingress
    weight: 30
product_name: voyager
menu_name: product_voyager_6.0.0-alpha.0
section_menu_id: guides
---

# CORS

Applying `ingress.kubenretes.io/enable-cors` annotation in ingress enables CORS for all HTTP Frontend.

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotation:
    ingress.kubernetes.io/enable-cors: 'true'
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

Applying the annotation in ingress will have the following effects, will add the CORS Header in the response.

```
$ curl -v -X 'GET' -k -H 'Origin: foo.bar.com' 'http://foo.bar.com'
 HTTP/1.1 200 OK
 Date: Mon, 02 Oct 2017 12:31:00 GMT
 Content-Length: 280
 Content-Type: text/plain; charset=utf-8
 Access-Control-Allow-Origin: foo.bar.com
 Access-Control-Allow-Methods: GET, HEAD, OPTIONS, POST, PUT
 Access-Control-Allow-Credentials: true
 Access-Control-Allow-Headers: Origin, Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers, Authorization

```