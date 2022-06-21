---
title: Securing Kubernetes Dashboard | Kubernetes Ingress
menu:
  docs_{{ .version }}:
    identifier: oauth2-dashboard
    name: Kubernetes Dashboard
    parent: oauth2-security
    weight: 20
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Securing Kubernetes Dashboard Using GitHub Oauth

In this example we will deploy kubernetes dashboard and access it through ingress. Also secure the access with voyager external auth using github as auth provider.

## Deploy Dashboard

```
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v1.8.3/src/deploy/recommended/kubernetes-dashboard.yaml
```

By default the dashboard configures HTTPS with a self signed certificate. We need to apply `ingress.appscode.com/backend-tls: ssl verify none` annotation to `kubernetes-dashboard` service so that haproxy pod can establish HTTPS connection with dashboard pod.

```
$ kubectl annotate service kubernetes-dashboard -n voyager ingress.appscode.com/backend-tls='ssl verify none'
```

## Configure GitHub Oauth App

Configure github auth provider by following instructions provided [here](https://github.com/bitly/oauth2_proxy#github-auth-provider) and generate client-id and client-secret.

Set `Authorization callback URL` to `https://<host:port>/oauth2/callback`. In this example it is set to `https://voyager.appscode.ninja`.

## Configure and Deploy Oauth Proxy

```yaml
$ kubectl apply -f oauth2-proxy.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
  namespace: voyager
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: oauth2-proxy
  template:
    metadata:
      labels:
        k8s-app: oauth2-proxy
    spec:
      containers:
      - args:
        - --provider=github
        - --email-domain=*
        - --upstream=file:///dev/null
        - --http-address=0.0.0.0:4180
        - --cookie-secure=true
        env:
        - name: OAUTH2_PROXY_CLIENT_ID
          value: ...
        - name: OAUTH2_PROXY_CLIENT_SECRET
          value: ...
        - name: OAUTH2_PROXY_COOKIE_SECRET
          value: ...
        image: quay.io/pusher/oauth2_proxy:v3.1.0
        imagePullPolicy: Always
        name: oauth2-proxy
        ports:
        - containerPort: 4180
          protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
  namespace: voyager
spec:
  ports:
  - name: http
    port: 4180
    protocol: TCP
    targetPort: 4180
  selector:
    k8s-app: oauth2-proxy
```

## Create TLS Secret

```bash
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt -subj "/CN=voyager.appscode.ninja"
$ kubectl create secret tls tls-secret --key /tmp/tls.key --cert /tmp/tls.crt -n voyager
```

## Deploy Ingress

```yaml
$ kubectl apply -f auth-ingress.yaml

apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: voyager
spec:
  tls:
  - secretName: tls-secret
    hosts:
    - voyager.appscode.ninja
  frontendRules:
  - port: 443
    auth:
      oauth:
      - host: voyager.appscode.ninja
        authBackend: auth-be
        authPath: /oauth2/auth
        signinPath: /oauth2/start
        paths:
        - /
  rules:
  - host: voyager.appscode.ninja
    http:
      paths:
      - path: /
        backend:
          service:
            name: kubernetes-dashboard
            port:
              number: 443
      - path: /oauth2
        backend:
          name: auth-be
          service:
            name: oauth2-proxy
            port:
              number: 4180
```

## Access DashBoard

Now browse https://voyager.appscode.ninja, it will redirect you to GitHub login page. After successful login, it will redirect you to dashboard login page.

We will use token of an existing service-account `replicaset-controller` to login dashboard. It should have permissions to see Replica Sets in the cluster. You can also create your own service-account with different roles.

```
$ kubectl describe serviceaccount -n voyager replicaset-controller

Name:                replicaset-controller
Namespace:           voyager
Labels:              <none>
Annotations:         <none>
Image pull secrets:  <none>
Mountable secrets:   replicaset-controller-token-b5mgw
Tokens:              replicaset-controller-token-b5mgw
Events:              <none>
```

```
$ kubectl describe secret replicaset-controller-token-b5mgw -n voyager

Name:         replicaset-controller-token-b5mgw
Namespace:    voyager
Labels:       <none>
Annotations:  kubernetes.io/service-account.name=replicaset-controller
              kubernetes.io/service-account.uid=b53b12b6-693c-11e8-9cb8-8ee164da275a

Type:  kubernetes.io/service-account-token

Data
====
ca.crt:     1006 bytes
namespace:  11 bytes
token:      ...
```

Now use the token to login dashboard.
