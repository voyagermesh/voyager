---
title: Configure Ingress Service Annotations
menu:
  docs_{{ .version }}:
    identifier: service-annotations-configuration
    name: Service Annotations
    parent: config-ingress
    weight: 10
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Service Annotations

You can specify annotations applied to HAProxy services through ingress annotation `ingress.appscode.com/annotations-service`. You have to provide it as a json formatted string to string map.

## Ingress Example

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/annotations-service: '{"foo": "bar", "bar":"foo"}'
spec:
  rules:
  - host: voyager.appscode.test
    http:
      paths:
      - path: /foo
        backend:
          service:
            name: test-server
            port:
              number: 80
```

It will add following annotations to HAProxy pods:

```yaml
annotations:
    bar: foo
    foo: bar
    ingress.appscode.com/last-applied-annotation-keys: bar,foo
```

```bash
$ kubectl get svc voyager-test-ingress -o=jsonpath='{.metadata.annotations}' | tr " " "\n"

map[foo:bar
bar:foo
ingress.appscode.com/last-applied-annotation-keys:foo,bar
ingress.appscode.com/origin-api-schema:voyager.appscode.com/v1
ingress.appscode.com/origin-name:test-ingress]
```