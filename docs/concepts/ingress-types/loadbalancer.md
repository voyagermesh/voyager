---
title: LoadBalancer Ingress | Voyager
menu:
  docs_{{ .version }}:
    identifier: loadbalancer-ingress
    name: LoadBalancer
    parent: ingress-types-concepts
    weight: 10
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: concepts
---

# LoadBalancer

In `LoadBalancer` type Ingress, HAProxy pods are exposed via a LoadBalancer type Kubernetes service named `voyager-${ingress-name}`. You can apply the `ingress.appscode.com/type: LoadBalancer` annotation on a Ingress object to enable this type of Ingress. This is also the default type for Ingress objects. So, this annotation is not required to enable this type.

## How It Works

- First, install Voyager operator in your cluster following the steps [here](/docs/setup/install.md).

- Now, deploy test servers using [this script](/docs/examples/ingress/types/loadbalancer/deploy-servers.sh) script.

```bash
curl -fsSL https://raw.githubusercontent.com/voyagermesh/voyager/{{< param "info.version" >}}/docs/examples/ingress/types/loadbalancer/deploy-servers.sh | bash

deployment "nginx" created
service "web" exposed
deployment "echoserver" created
service "rest" exposed
```

- Now, create an Ingress object running

```bash
kubectl apply -f https://raw.githubusercontent.com/voyagermesh/voyager/{{< param "info.version" >}}/docs/examples/ingress/types/loadbalancer/ing.yaml
```

Please note the annotation on ingress:

```yaml
  annotations:
    ingress.appscode.com/type: LoadBalancer
```

```bash
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

```bash
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

```bash
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

## FAQ

**How do I ensure that IP assigned my Ingress does not change?**

You can allocate a static IP to a LoadBalancer Ingress managed by Voyager. Say for example, you are using GKE. When you create an Ingress object, Voyager will create a Kubernetes Service of type LoadBalancer. This service will automatically get a regional IP. If you want to keep that IP, you can mark that IP as static in Google cloud console
and the apply the annotation to your Ingress.

```yaml
  annotations:
    ingress.appscode.com/type: LoadBalancer
 `  ingress.appscode.com/load-balancer-ip: 'a.b.c.d'`
```


**How do I use a Global Static IP (anycast IP) with an Ingress in GKE?**

You can't use Global Static IP with a LoabBalancer Ingress managed by GKE. Voyager creates a `LoadBalancer` Service to expose HAProxy pods. Under the hood, Kubernetes creates a `Network LoadBalancer` to expose that Kubernetes service. Network LoadBalancers can only use regional static IPs.

If you want to use Global static IP with Google Cloud, these pods need to be exposed via  a HTTP LoadBalancer. Voyager does not support this today. This is not a priority for us but if you want to contribute, we can talk more. To use HTTP LoadBalancers today, you can use the `gce` ingress controller: https://github.com/kubernetes/ingress-gce . You may already know that HTTP LoadBalancer can only open port 80, 8080 and 443 and serve HTTP traffic. Please consult the official docs for more details: https://cloud.google.com/compute/docs/load-balancing/


**How to use LoadBalancer type ingress in Openstack?**

If you need to create an internal LB in Openstack, you can do so using `ingress.appscode.com/annotations-service` annotation on the Ingress object.

```yaml
  annotations:
    ingress.appscode.com/type: LoadBalancer
    ingress.appscode.com/annotations-service: |
      {
        "service.beta.kubernetes.io/openstack-internal-load-balancer": "true"
      }
```


**How to use LoadBalancer type ingress in Minikube cluster?**

Minikube clusters do not support service type `LoadBalancer`. So, you can try the following workarounds:

- You can set the `Host` header is your http request to match the expected domain and port. This will ensure HAProxy matches the rules properly.

```bash
$ curl -vv <minikube-ip>:<node-port> -H "Host: app.example.com"
```

- This work around is available thanks to [@david92rl](https://github.com/david92rl). You can use a service type ClusterIP with an ip fixed (like 10.0.0.150), then create a route to it from host machine.

**_Minikube on Mac with virtualbox/vmware providers_**

```bash
sudo route -n delete ${K8S_NETWORK} > /dev/null 2>&1
sudo route -n add ${K8S_NETWORK} $(minikube ip)
interface=$(ifconfig 'bridge0' | grep member | awk '{print $2}' | xargs | awk '{print $1}')
sudo ifconfig bridge0 -hostfilter ${interface}
```

**_Minikube on Linux_**

```bash
sudo ip route del ${K8S_NETWORK}
sudo ip route add ${K8S_NETWORK} via $(minikube ip)
```

*K8S_NETWORK* usually is `10.0.0.0/24` but it's worth to double check always.
