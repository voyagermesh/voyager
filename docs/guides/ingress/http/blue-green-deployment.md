---
title: Blue Green Deployments | Kubernetes Ingress
menu:
  docs_{{ .version }}:
    identifier: blue-green-http
    name: Blue Green Deployment
    parent: http-ingress
    weight: 60
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Blue Green Deployments

Voyager supports Blue Green deployments using weighted loadbalancing for backend pods. Following example illustrates a weighted loadbalancer scenario.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: deployment
    app-version: v1
  name: deployment-1
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: deployment
      app-version: v1
  template:
    metadata:
      labels:
        app: deployment
        app-version: v1
      annotations:
          ingress.appscode.com/backend-weight: "90"
    spec:
      containers:
      - env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        image: appscode/test-server:1.1
        imagePullPolicy: IfNotPresent
        name: server
        ports:
        - containerPort: 8080
          name: http-1
          protocol: TCP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: deployment
    app-version: v2
  name: deployment-2
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: deployment
      app-version: v2
  template:
    metadata:
      labels:
        app: deployment
        app-version: v2
      annotations:
            ingress.appscode.com/backend-weight: "10"
    spec:
      containers:
      - env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        image: appscode/test-server:1.1
        imagePullPolicy: IfNotPresent
        name: server
        ports:
        - containerPort: 8080
          name: http-1
          protocol: TCP
```

Two different workload with the annotation `ingress.appscode.com/backend-weight` and one single service pointing to them

```yaml
apiVersion: v1
kind: Service
metadata:
  name: deployment-svc
  namespace: default
spec:
  ports:
  - name: http-1
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: deployment
```

The following ingress will forward 90% traffic to `deployment-1` and 10% to `deployment-2`

```yml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ing
  namespace: default
spec:
  rules:
  - http:
      paths:
      - backend:
          service:
            name: deployment-svc
            port:
              number: 80
        path: /testpath
```

The default [weight](http://cbonte.github.io/haproxy-dconv/1.8/configuration.html#weight) is 1, and the maximal value is 256. A value of 0 means the server will not participate in load-balancing but will still accept persistent connections.