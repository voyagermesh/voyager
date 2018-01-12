---
title: NodePort Ingress | Voyager
menu:
  product_voyager_5.0.0-rc.11:
    identifier: nodeport-ingress
    name: NodePort
    parent: ingress-types-concepts
    weight: 15
product_name: voyager
menu_name: product_voyager_5.0.0-rc.11
section_menu_id: concepts
---

# NodePort

In `NodePort` type Ingress, HAProxy pods are exposed via a NodePort type Kubernetes service named `voyager-${ingress-name}`. To enable this, apply the `ingress.appscode.com/type: NodePort` annotation on a Ingress object.

## How It Works

- First, deploy voyager operator.

```console
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.11/hack/deploy/voyager.sh \
    | bash -s -- --provider=minikube
```

- Now, deploy test servers using [this script](/docs/examples/ingress/types/nodeport/deploy-servers.sh) script.

```console
curl -fsSL https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.11/docs/examples/ingress/types/nodeport/deploy-servers.sh | bash

deployment "nginx" created
service "web" exposed
deployment "echoserver" created
service "rest" exposed
```

- Now, create an Ingress object running `kubectl apply -f https://raw.githubusercontent.com/appscode/voyager/5.0.0-rc.11/docs/examples/ingress/types/nodeport/ing.yaml`. Please note the annotaiton on ingress:

```yaml
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/force-service-port: "true"
```

```console
$ kubectl get pods,svc
NAME                                       READY     STATUS    RESTARTS   AGE
po/echoserver-848b75d85-hshcq              1/1       Running   0          2m
po/nginx-7c87f569d-gxftw                   1/1       Running   0          5m
po/voyager-test-ingress-55f9b58b8f-vpfmx   1/1       Running   0          1m

NAME                       CLUSTER-IP   EXTERNAL-IP   PORT(S)        AGE
svc/kubernetes             10.0.0.1     <none>        443/TCP        7m
svc/rest                   10.0.0.188   <none>        80/TCP         2m
svc/voyager-test-ingress   10.0.0.216   <nodes>       80:31656/TCP   1m
svc/web                    10.0.0.11    <none>        80/TCP         5m

$ minikube ip
192.168.99.100
```


```console
$ curl -vv 192.168.99.100:31656 -H "Host: web.example.com"
* Rebuilt URL to: 192.168.99.100:31656/
*   Trying 192.168.99.100...
* Connected to 192.168.99.100 (192.168.99.100) port 31656 (#0)
> GET / HTTP/1.1
> Host: web.example.com
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.13.6
< Date: Mon, 30 Oct 2017 11:17:43 GMT
< Content-Type: text/html
< Content-Length: 612
< Last-Modified: Thu, 14 Sep 2017 16:35:09 GMT
< ETag: "59baafbd-264"
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
* Connection #0 to host 192.168.99.100 left intact
```

```console
$ curl -vv 192.168.99.100:31656 -H "Host: app.example.com"
* Rebuilt URL to: 192.168.99.100:31656/
*   Trying 192.168.99.100...
* Connected to 192.168.99.100 (192.168.99.100) port 31656 (#0)
> GET / HTTP/1.1
> Host: app.example.com
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.10.0
< Date: Mon, 30 Oct 2017 11:17:51 GMT
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
user-agent=curl/7.47.0
x-forwarded-for=172.17.0.1
BODY:
* Connection #0 to host 192.168.99.100 left intact
```

## externalIP
If you are using NodePort type Ingress, you can specify [`externalIPs`](https://kubernetes.io/docs/concepts/services-networking/service/#external-ips) for the NodePort service used to export HAProxy pods.

If you are running Voyager as Ingress controller in your `baremetal` cluster and want to assing a VIP to the HAProxy service for a highly availble Ingress setup, you should use this option.

Below is an example Ingress yaml that shows how to specify externalIPs:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/force-service-port: "true"
spec:
  externalIPs:
  - a.b.c.d
  rules:
  - host: 'web.example.com'
    http:
      paths:
      - backend:
          serviceName: web
          servicePort: 80
        path: /
  - host: '*.example.com'
    http:
      paths:
      - backend:
          serviceName: rest
          servicePort: 80
        path: /
```

## Understanding `ingress.appscode.com/force-service-port` annotation

When you are using `NodePort` type Ingress, your extenral clients (example, web browser) can connect to your backend services in 2 ways:

 - Using the `NodePort` assigned to the HAProxy service. In this scenrios, external traffic directly hits the HAProxy NodePort service. So, HAProxy will see that incoming traffic is using Host like `domain:node-port`. To ensure that HAProxy matches against the NodePort, use the annotation `ingress.appscode.com/force-service-port: "false"` . This is also considered the the default value for this annotation. So, if you do not provide this annotation, it will be considered set to `false`. To ensure that HAProxy service ports stay fixed, define them in the Ingress YAML following: https://github.com/appscode/voyager/blob/master/docs/guides/ingress/configuration/node-port.md

 - You may expose the NodePort service using an external hardware loadbalancer (like F5) or software loadbalancer like ALB in AWS. In these scnerios, the front load balancer will receive connections on service port from external clients (like web browser) and connect to the HAProxy NodePorts. So, HAProxy will see that incoming traffic is using Host like `domain:ing-port`. To ensure that HAProxy matches against the NodePort, use the annotation `ingress.appscode.com/force-service-port: "true"` .
