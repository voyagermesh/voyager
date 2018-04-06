# External Authentication

This example demonstrates how to configure
[External Authentication](https://oauth.net/2/) on Voyager Ingress controller.

## Example using Github (no TLS)

First create a new github oauth app from [here](https://github.com/settings/applications) and generate client-id and client-secret.

Set Authorization callback URL to `http://<host:port>/oauth2`.

In this example it is set to `http://voyager.appscode.ninja:32666/oauth2`.

Now deploy and expose a test server:

```console
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
$ kubectl expose deployment test-server --port=80 --target-port=8080
```

Deploy and expose oauth2-proxy:

```yaml
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
        - --provider=github
        - --email-domain=*
        - --upstream=file:///dev/null
        - --http-address=0.0.0.0:4180
        - --cookie-secure=false
        env:
        - name: OAUTH2_PROXY_CLIENT_ID
          value: ...
        - name: OAUTH2_PROXY_CLIENT_SECRET
          value: ...
        - name: OAUTH2_PROXY_COOKIE_SECRET
          value: Y/XCgwGzcE/BIkhTtXFcSQ==
        image: docker.io/colemickens/oauth2_proxy:latest
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

Note that, for insecure/non-tls connections, we have to set `cookie-secure=false`.

Create ingress:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: "true"
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
      nodePort: 32666
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

Now browse followings:

- http://voyager.appscode.ninja:32666/app (external-auth required)
- http://voyager.appscode.ninja:32666/health (external-auth not required)

## Example using Github (with TLS)

First create a new github oauth app from [here](https://github.com/settings/applications) and generate client-id and client-secret.

Set Authorization callback URL to `https://<host:port>/oauth2`.

In this example it is set to `https://voyager.appscode.ninja:32666/oauth2`.

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

Deploy and expose oauth2-proxy:

```yaml
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
          value: Y/XCgwGzcE/BIkhTtXFcSQ==
        image: docker.io/colemickens/oauth2_proxy:latest
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

Note that, for secure/tls connections, we have to set `cookie-secure=true`.

Create ingress:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: "true"
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
      nodePort: 32666
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

Now browse followings:

- https://voyager.appscode.ninja:32666/app (external-auth required)
- https://voyager.appscode.ninja:32666/health (external-auth not required)