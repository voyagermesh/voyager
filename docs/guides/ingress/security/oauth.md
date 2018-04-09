# External Authentication

You can configure [external authentication / oauth](https://oauth.net/2/) on Voyager Ingress controller via `frontendrules`.
For this you have to configure and expose [oauth2-proxy](https://github.com/bitly/oauth2_proxy) and specify it as a backend under same host.

This example will demonstrate how to configure external authentication in both TLS and non-TLS mode using `github` as auth provider.

## Example using Github (non-TLS)

First create a new github oauth app from [here](https://github.com/settings/applications) and generate client-id and client-secret.
Set Authorization callback URL to `http://<host:port>/oauth2`. 
In this example it is set to `http://voyager.appscode.ninja:32666/oauth2`.

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

Finally create the ingress:

```yaml
$ kubectl apply -f auth-ingress.yaml

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

Now browse the followings:

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

Finally create the ingress:

```yaml
$ kubectl apply -f auth-ingress.yaml

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

Now browse the followings:

- https://voyager.appscode.ninja:32666/app (external-auth required)
- https://voyager.appscode.ninja:32666/health (external-auth not required)

Please note the followings:

- Oauth will be enabled only for the specified paths. It is not necessary that this paths should match with the paths specified in the http-rules.

- Auth backend and app backend should be under same host.

- For secure/tls connections, you have to set `cookie-secure=true` (default) and for insecure/non-tls connections, you have to set `cookie-secure=false` in `oauth2-proxy`.

- You can use any random string as `OAUTH2_PROXY_COOKIE_SECRET`. You can generate one using following command:

```console
$ python -c 'import os,base64; print base64.b64encode(os.urandom(16))'
```
 
- If you use standard ports, you have to write frontend rules under port `80` for non-tls and under port `443` for tls.

- You can configure different auth backends for different paths under same host. For example:

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
        authBackend: google-auth
        authPath: /google/auth
        signinPath: /google/start
        paths:
        - /foo
      - host: voyager.appscode.ninja
        authBackend: github-auth
        authPath: /github/auth
        signinPath: /github/start
        paths:
        - /bar
  rules:
  - host: voyager.appscode.ninja
    http:
      nodePort: 32666
      paths:
      - path: /health
        backend:
          serviceName: test-server
          servicePort: 80
      - path: /foo
        backend:
          serviceName: test-server
          servicePort: 80
      - path: /bar
        backend:
          serviceName: test-server
          servicePort: 80
      - path: /google
        backend:
          name: google-auth
          serviceName: oauth2-proxy-google
          servicePort: 4180
      - path: /github
        backend:
          name: github-auth
          serviceName: oauth2-proxy-github
          servicePort: 4180
```