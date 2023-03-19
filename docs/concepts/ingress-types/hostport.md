---
title: HostPort Ingress | Voyager
menu:
  docs_{{ .version }}:
    identifier: hostport-ingress
    name: HostPort
    parent: ingress-types-concepts
    weight: 20
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: concepts
---

# HostPort

In `HostPort` type Ingress, HAProxy pods are run via a Kubernetes deployment named `voyager-${ingress-name}` with `hostNetwork: true`. A headless Service is also created for the HAProxy pods. To enable this, apply the `ingress.appscode.com/type: HostPort` annotation on a Ingress object.

## How It Works

- First, install Voyager operator in your cluster following the steps [here](/docs/setup/install.md).

- Now, deploy test servers using [this script](/docs/examples/ingress/types/hostport/deploy-servers.sh) script.

```bash
curl -fsSL https://raw.githubusercontent.com/voyagermesh/voyager/{{< param "info.version" >}}/docs/examples/ingress/types/hostport/deploy-servers.sh | bash

deployment "nginx" created
service "web" exposed
deployment "echoserver" created
service "rest" exposed
```

- Now, create an Ingress object running

```bash
kubectl apply -f https://raw.githubusercontent.com/voyagermesh/voyager/{{< param "info.version" >}}/docs/examples/ingress/types/hostport/ing.yaml
```

Please note the annotation on ingress:

```yaml
  annotations:
    ingress.appscode.com/type: HostPort
```

```bash
$ kubectl get pods,svc
NAME                                       READY     STATUS    RESTARTS   AGE
po/echoserver-566fcc4fdb-7fth7             1/1       Running   0          6m
po/nginx-d5dc44cf7-m4xcg                   1/1       Running   0          6m
po/voyager-test-ingress-668594cc46-5zswh   1/1       Running   0          4m

NAME                       TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
svc/kubernetes             ClusterIP   10.96.0.1      <none>        443/TCP   1h
svc/rest                   ClusterIP   10.103.13.42   <none>        80/TCP    6m
svc/voyager-test-ingress   ClusterIP   None           <none>        80/TCP    4m
svc/web                    ClusterIP   10.99.232.60   <none>        80/TCP    6m
```

- Now, ssh into the minikube vm and run the following commands from host:

```bash
$ minikube ssh

$ curl -vv 127.0.0.1 -H "Host: web.example.com"
> GET / HTTP/1.1
> Host: web.example.com
> User-Agent: curl/7.53.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.13.8
< Date: Thu, 28 Dec 2017 04:27:20 GMT
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
$ curl -vv 127.0.0.1 -H "Host: app.example.com"
> GET / HTTP/1.1
> Host: app.example.com
> User-Agent: curl/7.53.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx/1.10.0
< Date: Thu, 28 Dec 2017 04:27:39 GMT
< Content-Type: text/plain
< Transfer-Encoding: chunked
<
CLIENT VALUES:
client_address=172.17.0.1
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
x-forwarded-for=127.0.0.1
BODY:
```

Now, if you run `netstat`, you should port 80 is listened on by haproxy.

```bash
$ netstat -tuln | grep 80
tcp        0      0 0.0.0.0:80              0.0.0.0:*               LISTEN
tcp        0      0 127.0.0.1:2380          0.0.0.0:*               LISTEN
```

## FAQ

## Does Voyager configure firewalls for HostPort Ingress?

Voyager operator will configure firewall rules for HostPort Ingress for the following cloud providers: AWS, GCE/GKE .

## What IAM permissions are required for Voyager operator to configure firewalls for HostPort Ingress in AWS?

 - Master: For aws clusters provisioned via [Kops](https://github.com/kubernetes/kops/blob/master/docs/iam_roles.md), no additional permission should be needed. Master instances already has `ec2:*` iam permissions.

- Nodes: `Describe*` permissions are applied by default. Additional `write` permissions need to be applied are:
```
{
  "Effect": "Allow",
  "Action": [
	"ec2:AuthorizeSecurityGroupIngress",
	"ec2:CreateRoute",
	"ec2:CreateSecurityGroup",
	"ec2:CreateTags",
	"ec2:DeleteRoute",
	"ec2:DeleteSecurityGroup",
	"ec2:ModifyInstanceAttribute",
	"ec2:RevokeSecurityGroupIngress",
	"ec2:DescribeInstances",
	"ec2:DescribeRouteTables",
	"ec2:DescribeSecurityGroups",
	"ec2:DescribeSubnets"
  ],
  "Resource": "*"
}
```
