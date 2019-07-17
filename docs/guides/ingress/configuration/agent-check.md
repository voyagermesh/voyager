---
title: Agent Check | Kubernetes Ingress
menu:
  product_voyager_10.0.0:
    identifier: agent-check
    name: Agent Check
    parent: config-ingress
    weight: 20
product_name: voyager
menu_name: product_voyager_10.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Agent Check

[haproxy-agent-check](http://cbonte.github.io/haproxy-dconv/1.8/configuration.html#5.2-agent-check) can be enabled for a specific backend server by assigning the agent server port in `ingress.appscode.com/agent-port` annotations to the corresponding service. You can also add [agent-inter](http://cbonte.github.io/haproxy-dconv/1.8/configuration.html#agent-inter) in `ingress.appscode.com/agent-interval` annotations to the same service, which defaults to 2000ms if not mentioned.

## Example

First deploy and expose a test server:

```yaml
$ kubectl apply -f test-server.yaml

apiVersion: apps/v1
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
      - image: appscode/test-server:2.4
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
  - port: 5555
    targetPort: 5555
    name: agent
  selector:
    run: test-server
```

Here, port 8080 will serve client's request and port 5555 will be used for agent-check.

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

Now we need to annotate the backend service to enable agent-check for that backend.

```console
$ kubectl annotate svc test-server ingress.appscode.com/agent-port="5555"
```

To change the default agent-interval value, annotate the same service with:
```console
$ kubectl annotate svc test-server ingress.appscode.com/agent-interval="3s"
```

## Time Format

These timeout values are generally expressed in milliseconds (unless explicitly stated
otherwise) but may be expressed in any other unit by suffixing the unit to the
numeric value. Supported units are :

- us : microseconds. 1 microsecond = 1/1000000 second
- ms : milliseconds. 1 millisecond = 1/1000 second. This is the default.
- s  : seconds. 1s = 1000ms
- m  : minutes. 1m = 60s = 60000ms
- h  : hours.   1h = 60m = 3600s = 3600000ms
- d  : days.    1d = 24h = 1440m = 86400s = 86400000ms

