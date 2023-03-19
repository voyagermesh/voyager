---
title: Internal Ingress | Voyager
menu:
  docs_{{ .version }}:
    identifier: internal-ingress
    name: Internal
    parent: ingress-types-concepts
    weight: 25
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: concepts
---

# Internal

In `Internal` type Ingress, HAProxy pods are exposed via a ClusterIP type Kubernetes service named `voyager-${ingress-name}`. To enable this, apply the `ingress.appscode.com/type: Internal` annotation on a Ingress object. Unlike Kubernetes Service which operates at L4 level, this creates a cluster internal L7 proxy. An example use-case is proxy for ElasticSearch cluster to handle persistent connections, alleviating the ElasticSearch servers from having to deal w/ tons of connection creations.

## How It Works

- First, install Voyager operator in your cluster following the steps [here](/docs/setup/install.md).

- Now, deploy test servers using [this script](/docs/examples/ingress/types/internal/deploy-servers.sh) script.

```bash
curl -fsSL https://raw.githubusercontent.com/voyagermesh/voyager/{{< param "info.version" >}}/docs/examples/ingress/types/internal/deploy-servers.sh | bash

deployment "nginx" created
service "web" exposed
deployment "echoserver" created
service "rest" exposed
```

- Now, create an Ingress object running

```bash
kubectl apply -f https://raw.githubusercontent.com/voyagermesh/voyager/{{< param "info.version" >}}/docs/examples/ingress/types/internal/ing.yaml
```

Please note the annotation on ingress:

```yaml
  annotations:
    ingress.appscode.com/type: Internal
```

```bash
$ kubectl get pods,svc
NAME                                       READY     STATUS    RESTARTS   AGE
po/echoserver-566fcc4fdb-bw4xm             1/1       Running   0          1m
po/nginx-d5dc44cf7-brgd4                   1/1       Running   0          1m
po/voyager-test-ingress-6859dc5ddd-9mmz2   1/1       Running   0          34s

NAME                       TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
svc/kubernetes             ClusterIP   10.96.0.1       <none>        443/TCP   7m
svc/rest                   ClusterIP   10.105.80.152   <none>        80/TCP    1m
svc/voyager-test-ingress   ClusterIP   10.97.153.185   <none>        80/TCP    34s
svc/web                    ClusterIP   10.96.37.26     <none>        80/TCP    1m

$ minikube ip
192.168.99.100
```

- Now, ssh into the minikube vm and run the following commands from host:

```bash
$ minikube ssh

$ curl -vv 10.97.153.185 -H "Host: web.example.com"
> GET / HTTP/1.1
> Host: web.example.com
> User-Agent: curl/7.53.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.13.8
< Date: Thu, 18 Jan 2018 06:58:29 GMT
< Content-Type: text/html
< Content-Length: 612
< Last-Modified: Tue, 26 Dec 2017 11:11:22 GMT
< ETag: "5a422e5a-264"
< Accept-Ranges: bytes
<
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
```

```bash
$ curl -vv 10.97.153.185 -H "Host: app.example.com"
> GET / HTTP/1.1
> Host: app.example.com
> User-Agent: curl/7.53.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.10.0
< Date: Thu, 18 Jan 2018 06:59:05 GMT
< Content-Type: text/plain
< Transfer-Encoding: chunked
<
CLIENT VALUES:
client_address=172.17.0.7
command=GET
real path=/
query=nil
request_version=1.1
request_uri=http://app.example.com:8080/

SERVER VALUES:
server_version=nginx: 1.10.0 - lua: 10001

HEADERS RECEIVED:
accept=*/*
connection=close
host=app.example.com
user-agent=curl/7.53.0
x-forwarded-for=10.0.2.15
BODY:
-no body in request-
```
