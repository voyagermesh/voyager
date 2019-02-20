---
title: OAuth2 Authentication | Kubernetes Ingress
menu:
  product_voyager_9.0.0:
    identifier: oauth2-google
    name: OAuth2 Google
    parent: oauth2-security
    weight: 20
product_name: voyager
menu_name: product_voyager_9.0.0
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# OAuth2 Authentication Using Google

This example will demonstrate how to configure external authentication in both TLS and non-TLS mode using Google as auth provider.

## Example using Google (no TLS)

First configure google auth provider by following instructions provided [here](https://github.com/bitly/oauth2_proxy#google-auth-provider) and generate client-id and client-secret.

In this example `Authorized JavaScript origins` is set to `http://voyager.appscode.ninja`
and `Authorized redirect URIs` is set to `http://voyager.appscode.ninja/oauth2/callback`.

Now deploy and expose a test server:

```console
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
$ kubectl expose deployment test-server --port=80 --target-port=8080
```

Configure, deploy and expose oauth2-proxy:

```yaml
$ kubectl apply -f oauth2-proxy.yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
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
        - --provider=google
        - --email-domain=*
        - --upstream=file:///dev/null
        - --http-address=0.0.0.0:4180
        - --cookie-secure=false
        - --pass-id-token=true
        - --set-xauthrequest=true
        env:
        - name: OAUTH2_PROXY_CLIENT_ID
          value: ...
        - name: OAUTH2_PROXY_CLIENT_SECRET
          value: ...
        - name: OAUTH2_PROXY_COOKIE_SECRET
          value: ...
        image: appscode/oauth2_proxy:2.3.1
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
spec:
  ports:
  - name: http
    port: 4180
    protocol: TCP
    targetPort: 4180
  selector:
    k8s-app: oauth2-proxy
```

Here, `--set-xauthrequest` flag sets `X-Auth-Request-User` and `X-Auth-Request-Email` headers, which will be forwarded to backend. It also sets `X-Auth-Request-Id-Token` header when `--pass-id-token` flag is `true`.

Finally create the ingress:

```yaml
$ kubectl apply -f auth-ingress.yaml

apiVersion: voyager.appscode.com/v1beta1
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
          serviceName: test-server
          servicePort: 80
      - path: /app
        backend:
          serviceName: test-server
          servicePort: 80
      - path: /oauth2
        backend:
          name: auth-be
          serviceName: oauth2-proxy
          servicePort: 4180
```

Now browse the followings:

- http://voyager.appscode.ninja/app (external-auth required)
- http://voyager.appscode.ninja/health (external-auth not required)

## Example using Google (with TLS)

First configure google auth provider by following instructions provided [here](https://github.com/bitly/oauth2_proxy#google-auth-provider) and generate client-id and client-secret.

In this example `Authorized JavaScript origins` is set to `https://voyager.appscode.ninja`
and `Authorized redirect URIs` is set to `https://voyager.appscode.ninja/oauth2/callback`.

Now deploy and expose a test server:

```console
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
$ kubectl expose deployment test-server --port=80 --target-port=8080
```

Create TLS secret:

```console
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt -subj "/CN=voyager.appscode.ninja"
$ kubectl create secret tls tls-secret --key /tmp/tls.key --cert /tmp/tls.crt
```

Configure, deploy and expose oauth2-proxy:

```yaml
$ kubectl apply -f oauth2-proxy.yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
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
        - --provider=google
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
        image: appscode/oauth2_proxy:2.3.1
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

apiVersion: voyager.appscode.com/v1beta1
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
          serviceName: test-server
          servicePort: 80
      - path: /app
        backend:
          serviceName: test-server
          servicePort: 80
      - path: /oauth2
        backend:
          name: auth-be
          serviceName: oauth2-proxy
          servicePort: 4180
```

Now browse the followings:

- https://voyager.appscode.ninja/app (external-auth required)
- https://voyager.appscode.ninja/health (external-auth not required)