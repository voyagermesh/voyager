# Securing Kubernetes Dashboard Using Github Oauth

In this example we will deploy kubernetes dashboard and access it through ingress.
Also secure the access with voyager external auth using github as auth provider.

## Deploy Dashboard

```
$ kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/tree/oauth-doc/docs/examples/ingress/oauth/dashboard.yaml
```

By default the dashboard configures HTTPS with a self signed certificate.
But since we will access dashboard through ingress without directly exposing it, we can use HTTP connection between haproxy and dashboard.
In order to do so we need following changes to the recommended setup described [here](https://raw.githubusercontent.com/kubernetes/dashboard/v1.8.3/src/deploy/recommended/kubernetes-dashboard.yaml).

- Set `livenessProbe` scheme to HTTP instead of HTTPS.
- Removed `auto-generate-certificates` flag and added `insecure-bind-address=0.0.0.0`, `insecure-port=9090`, `enable-insecure-loginDeploy` flags.
- Replaced port `8443` with `9090` and `443` with `80`.

## Configure Github Oauth App

Configure github auth provider by following instructions provided [here](https://github.com/bitly/oauth2_proxy#github-auth-provider) and generate client-id and client-secret.

Set `Authorization callback URL` to `https://<host:port>/oauth2/callback`.
In this example it is set to `https://voyager.appscode.ninja`.

## Configure and Deploy Oauth Proxy

```yaml
$ kubectl apply -f oauth2-proxy.yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    k8s-app: oauth2-proxy
  name: oauth2-proxy
  namespace: kube-system
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
        - OAUTH2_PROXY_COOKIE_SECRET = ...
        image: appscode/oauth2_proxy:2.2.0
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
  namespace: kube-system
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

```console
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt -subj "/CN=voyager.appscode.ninja"
$ kubectl create secret tls tls-secret --key /tmp/tls.key --cert /tmp/tls.crt -n kube-system
```

## Deploy Ingress

```yaml
$ kubectl apply -f auth-ingress.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: kube-system
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
          serviceName: kubernetes-dashboard
          servicePort: 80
      - path: /oauth2
        backend:
          name: auth-be
          serviceName: oauth2-proxy
          servicePort: 4180
```

## Access DashBoard

Now browse https://voyager.appscode.ninja, it will redirect you to Github login page.
After successful login, it will redirect you to dashboard login page.

We will use token of an existing service-account `replicaset-controller` to login dashboard. It should have permissions to see Replica Sets in the cluster. You can also create your own service-account with different roles.

```
$ kubectl describe serviceaccount -n kube-system replicaset-controller

Name:                replicaset-controller
Namespace:           kube-system
Labels:              <none>
Annotations:         <none>
Image pull secrets:  <none>
Mountable secrets:   replicaset-controller-token-b5mgw
Tokens:              replicaset-controller-token-b5mgw
Events:              <none>
```

```
$ kubectl describe secret replicaset-controller-token-b5mgw -n kube-system

Name:         replicaset-controller-token-b5mgw
Namespace:    kube-system
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

## References

- https://github.com/kubernetes/dashboard
- https://blog.heptio.com/on-securing-the-kubernetes-dashboard-16b09b1b7aca
