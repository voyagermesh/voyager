---
title: Forward Traffic to StatefulSet Pods | Kubernetes Ingress
menu:
  product_voyager_7.3.0:
    identifier: statefulset-http
    name: Statefulset Pod
    parent: http-ingress
    weight: 50
product_name: voyager
menu_name: product_voyager_7.3.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Forward Traffic to StatefulSet

## Forward Traffic to all Pods of a StatefulSet

There is the usual way of forwarding traffic to a Service matching a StatefulSet. Create a Service with the pods label selector as
selector, and use the service name as Backend ServiceName.

```yaml
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: http
spec:
  serviceName: "nginx-set"
  replicas: 2
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: gcr.io/google_containers/nginx-slim:0.8
        ports:
        - containerPort: 80
          name: http
----
apiVersion: v1
kind: Service
metadata:
  name: nginx-set
  labels:
    app: nginx
spec:
  ports:
  - port: 80
    name: http
  clusterIP: None
  selector:
    app: nginx
```

Create another service for StatefulSets pods with selector.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  labels:
    app: nginx
spec:
  ports:
  - port: 80
    name: http
  selector:
    app: nginx

```

And Use the service in the ingress Backend service name, as:

```yaml
backend:
  serviceName: nginx-service
  servicePort: '80'
```

That will forward traffic to your StatefulSets Pods.


## Forward Traffic to specific Pods of a StatefulSet
There is a way to send traffic to all or specific pod of a StatefulSet using voyager. You can set `hostNames` field in `Backend`, traffic will only forwarded to those pods.

For Example the above StatefulSet will create two pod.
```
web-0
web-1
```
Those are the host names.

Now Create a ingress that will only forward traffic to web-0
```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - path: '/testPath'
        backend:
          hostNames:
          - web-0
          serviceName: nginx-set #! There is no extra service. This
          servicePort: '80'      # is the Statefulset's Headless Service
```

Viola. Now all `/testPath` traffic will be sent to pod web-0 only. There is no extra service also.
The StatefulSet's Headless Service is enough. By using all the hostNames You can forward traffic to all pods.
