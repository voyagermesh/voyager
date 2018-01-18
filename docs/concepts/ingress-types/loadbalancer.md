---
title: LoadBalancer Ingress | Voyager
menu:
  product_voyager_5.0.0-rc.11:
    identifier: loadbalancer-ingress
    name: LoadBalancer
    parent: ingress-types-concepts
    weight: 10
product_name: voyager
menu_name: product_voyager_5.0.0-rc.11
section_menu_id: concepts
---

# LoadBalancer

In `LoadBalancer` type Ingress, HAProxy pods are exposed via a LoadBalancer type Kubernetes service named `voyager-${ingress-name}`. You can apply the `ingress.appscode.com/type: LoadBalancer` annotation on a Ingress object to enable this type of Ingress. This is also the default type for Ingress objects. So, this annotaion is not required to enable this type.

## How It Works

- First, deploy voyager operator.

```console
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.11/hack/deploy/voyager.sh \
    | bash -s -- --provider=gke
```

- Now, deploy test servers using [this script](/docs/examples/ingress/types/loadbalancer/deploy-servers.sh) script.

```console
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.11/docs/examples/ingress/types/loadbalancer/deploy-servers.sh | bash

deployment "nginx" created
service "web" exposed
deployment "echoserver" created
service "rest" exposed
```

- Now, create an Ingress object running `kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.11/docs/examples/ingress/types/loadbalancer/ing.yaml`. Please note the annotaiton on ingress:

```yaml
  annotations:
    ingress.appscode.com/type: LoadBalancer
```

```console
$ kubectl get pods,svc
NAME                                       READY     STATUS    RESTARTS   AGE
po/echoserver-848b75d85-wxdrz              1/1       Running   0          2m
po/nginx-7c87f569d-5q5mf                   1/1       Running   0          3m
po/voyager-test-ingress-687d6b7c65-qjqzt   1/1       Running   0          1m

NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)        AGE
svc/kubernetes             ClusterIP      10.11.240.1     <none>           443/TCP        4m
svc/rest                   ClusterIP      10.11.252.242   <none>           80/TCP         2m
svc/voyager-test-ingress   LoadBalancer   10.11.248.185   35.226.114.148   80:30854/TCP   1m
svc/web                    ClusterIP      10.11.253.33    <none>           80/TCP         2m
```

```console
$ curl -vv 35.226.114.148 -H "Host: web.example.com"
* Rebuilt URL to: 35.226.114.148/
*   Trying 35.226.114.148...
* Connected to 35.226.114.148 (35.226.114.148) port 80 (#0)
> GET / HTTP/1.1
> Host: web.example.com
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.13.8
< Date: Thu, 18 Jan 2018 06:40:49 GMT
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
* Connection #0 to host 35.226.114.148 left intact
```

```console
$ curl -vv 35.226.114.148 -H "Host: app.example.com"
* Rebuilt URL to: 35.226.114.148/
*   Trying 35.226.114.148...
* Connected to 35.226.114.148 (35.226.114.148) port 80 (#0)
> GET / HTTP/1.1
> Host: app.example.com
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.10.0
< Date: Thu, 18 Jan 2018 06:41:36 GMT
< Content-Type: text/plain
< Transfer-Encoding: chunked
<
CLIENT VALUES:
client_address=10.8.0.14
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
user-agent=curl/7.47.0
x-forwarded-for=10.8.0.1
BODY:
* Connection #0 to host 35.226.114.148 left intact
```
