---
title: OAuth2 Authentication Using GitHub | Kubernetes Ingress
menu:
  docs_{{ .version }}:
    identifier: oauth2-github
    name: OAuth2 GitHub
    parent: oauth2-security
    weight: 20
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# OAuth2 Authentication Using GitHub

This example will demonstrate how to configure external authentication in both TLS and non-TLS mode using GitHub as auth provider.

## Example using GitHub (no TLS)

First configure github auth provider by following instructions provided [here](https://github.com/bitly/oauth2_proxy#github-auth-provider) and generate client-id and client-secret.

Set `Authorization callback URL` to `http://<host:port>/oauth2/callback`.
In this example it is set to `http://voyager.appscode.ninja`.

Now deploy and expose a test server:

```bash
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
$ kubectl expose deployment test-server --port=80 --target-port=8080
```

Configure, deploy and expose oauth2-proxy:

```yaml
$ kubectl apply -f oauth2-proxy.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
  namespace: default
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
        - --cookie-secure=false
        - --set-xauthrequest=true
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
  namespace: default
spec:
  ports:
  - name: http
    port: 4180
    protocol: TCP
    targetPort: 4180
  selector:
    k8s-app: oauth2-proxy
```

Here, `--set-xauthrequest` flag sets `X-Auth-Request-User` and `X-Auth-Request-Email` headers, which will be forwarded to backend.

Finally create the ingress:

```yaml
$ kubectl apply -f auth-ingress.yaml

apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: default
spec:
  frontendRules:
  - port: 80
    auth:
      oauth:
      - host: voyager.appscode.ninja
        authBackend: auth-be
        authPath: /oauth2/auth
        signinPath: /oauth2/start
        paths: 
        - /app
  rules:
  - host: voyager.appscode.ninja
    http:
      paths:
      - path: /health
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /app
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /oauth2
        backend:
          name: auth-be
          service:
            name: oauth2-proxy
            port:
              number: 4180
```

Now browse the followings:

- http://voyager.appscode.ninja/app (external-auth required)
- http://voyager.appscode.ninja/health (external-auth not required)

## Example using GitHub (with TLS)

First configure github auth provider by following instructions provided [here](https://github.com/bitly/oauth2_proxy#github-auth-provider) and generate client-id and client-secret.

Set `Authorization callback URL` to `https://<host:port>/oauth2/callback`.
In this example it is set to `https://voyager.appscode.ninja`.

Now deploy and expose a test server:

```bash
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
$ kubectl expose deployment test-server --port=80 --target-port=8080
```

Create TLS secret:

```bash
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt -subj "/CN=voyager.appscode.ninja"
$ kubectl create secret tls tls-secret --key /tmp/tls.key --cert /tmp/tls.crt
```

Configure, deploy and expose oauth2-proxy:

```yaml
$ kubectl apply -f oauth2-proxy.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
  namespace: default
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
        - --set-xauthrequest=true
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
  namespace: default
spec:
  ports:
  - name: http
    port: 4180
    protocol: TCP
    targetPort: 4180
  selector:
    k8s-app: oauth2-proxy
```

Finally create the ingress:

```yaml
$ kubectl apply -f auth-ingress.yaml

apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: default
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
        - /app
  rules:
  - host: voyager.appscode.ninja
    http:
      paths:
      - path: /health
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /app
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /oauth2
        backend:
          name: auth-be
          service:
            name: oauth2-proxy
            port:
              number: 4180
```

Now browse the followings:

- https://voyager.appscode.ninja/app (external-auth required)
- https://voyager.appscode.ninja/health (external-auth not required)
