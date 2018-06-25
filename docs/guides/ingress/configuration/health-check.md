---
title: Backend Health Check | Kubernetes Ingress
menu:
  product_voyager_7.2.0:
    identifier: health-check
    name: Backend Health Check
    parent: config-ingress
    weight: 100
product_name: voyager
menu_name: product_voyager_7.2.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Server Health Check

You can enable [haproxy-health-checks](https://www.haproxy.com/documentation/aloha/7-0/traffic-management/lb-layer7/health-checks/) for a specific backend server by applying `ingress.appscode.com/check` and `ingress.appscode.com/check-port` annotations to the corresponding service. You can also configure health-check behavior using backend rules.

## Example

First deploy and expose a test server:

```yaml
$ kubectl apply -f test-server.yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    run: test-server
  name: test-server
  namespace: default
spec:
  selector:
    matchLabels:
      run: test-server
  template:
    metadata:
      labels:
        run: test-server
    spec:
      containers:
      - image: appscode/test-server:2.2
        name: test-server
---
apiVersion: v1
kind: Service
metadata:
  labels:
    run: test-server
  name: test-server
  namespace: default
spec:
  ports:
  - port: 8080
    targetPort: 8080
    name: web
  - port: 9090
    targetPort: 9090
    name: health
  selector:
    run: test-server
```

Here, port 8080 will serve client's request and port 9090 will be used for health checks.

Then deploy the ingress:

```yaml
$ kubectl apply test-ingress.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - http:
      paths:
      - path: /app
        backend:
          serviceName: test-server
          servicePort: 8080
```

Now we need to annotate the backend service to enable health check for that backend.

```console
$ kubectl annotate svc test-server ingress.appscode.com/check="true"
$ kubectl annotate svc test-server ingress.appscode.com/check-port="9090"
```

You can also specify the health-check behaviour using backend rules. For example:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - http:
      paths:
      - path: /app
        backend:
          serviceName: test-server
          servicePort: 8080
          backendRules:
          - 'option httpchk GET /testpath/ok'
          - 'http-check expect rstring (testpath/ok)'
```

