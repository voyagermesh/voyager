---
title: NodePort Ingress | Voyager
menu:
  docs_{{ .version }}:
    identifier: nodeport-ingress
    name: NodePort
    parent: ingress-types-concepts
    weight: 15
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: concepts
---

# NodePort

In `NodePort` type Ingress, HAProxy pods are exposed via a NodePort type Kubernetes service named `voyager-${ingress-name}`. To enable this, apply the `ingress.appscode.com/type: NodePort` annotation on a Ingress object.

## How It Works

- First, install Voyager operator in your cluster following the steps [here](/docs/setup/install.md).

- Then, deploy and expose a test server.

```bash
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
deployment "test-server" created

$ kubectl expose deployment test-server --type=LoadBalancer --port=80 --target-port=8080
service "test-server" exposed
```

- Now, create an Ingress with `ingress.appscode.com/type: NodePort` annotation.

```yaml
$ kubectl apply -f test-ingress.yaml

apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
spec:
  rules:
  - host: voyager.appscode.test
    http:
      paths:
      - path: /foo
        backend:
          service:
            name: test-server
            port:
              number: 80
```

```bash
$ kubectl get pods,svc
NAME                                       READY     STATUS    RESTARTS   AGE
po/test-server-68ddc845cd-x7dtv            1/1       Running   0          4h
po/voyager-test-ingress-5b758664f6-4vptr   1/1       Running   0          1m

NAME                       TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
svc/kubernetes             ClusterIP      10.96.0.1        <none>        443/TCP        2d
svc/test-server            LoadBalancer   10.105.13.31     <pending>     80:30390/TCP   21h
svc/voyager-test-ingress   NodePort       10.107.182.219   <none>        80:30800/TCP   39m
```

```bash
$ minikube service --url voyager-test-ingress
http://192.168.99.100:30800
```

```bash
$ curl -v -H 'Host: voyager.appscode.test' 192.168.99.100:30800/foo
*   Trying 192.168.99.100...
* Connected to 192.168.99.100 (192.168.99.100) port 30800 (#0)
> GET /foo HTTP/1.1
> Host: voyager.appscode.test
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Wed, 14 Feb 2018 09:34:30 GMT
< Content-Type: text/plain
< Transfer-Encoding: chunked
< Server: echoserver
<


Hostname: test-server-68ddc845cd-x7dtv

Pod Information:
	-no pod information available-

Server values:
	server_version=nginx: 1.13.3 - lua: 10008

Request Information:
	client_address=172.17.0.6
	method=GET
	real path=/foo
	query=
	request_version=1.1
	request_uri=http://voyager.appscode.test:8080/foo

Request Headers:
	accept=*/*
	connection=close
	host=voyager.appscode.test
	user-agent=curl/7.47.0
	x-forwarded-for=172.17.0.1

Request Body:
	-no body in request-

* Connection #0 to host 192.168.99.100 left intact
```

## External IP
If you are using NodePort type Ingress, you can specify [`externalIPs`](https://kubernetes.io/docs/concepts/services-networking/service/#external-ips) for the NodePort service used to export HAProxy pods.

If you are running Voyager as Ingress controller in your `baremetal` cluster and want to assing a VIP to the HAProxy service for a highly available Ingress setup, you should use this option.

Below is an example Ingress yaml that shows how to specify externalIPs:

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
spec:
  externalIPs:
  - a.b.c.d
  rules:
  - host: voyager.appscode.test
    http:
      paths:
      - path: /foo
        backend:
          service:
            name: test-server
            port:
              number: 80
```

## Understanding `ingress.appscode.com/use-node-port` annotation

When you are using `NodePort` type Ingress, your external clients (example, web browser) can connect to your backend services in 2 ways:

 - You may expose the NodePort service using an external hardware loadbalancer (like F5) or software loadbalancer like ALB in AWS. In these scenarios, the front loadbalancer will receive connections on service port from external clients (like web browser) and connect to the HAProxy NodePorts. So, HAProxy will see that incoming traffic is using Host like `domain:ing-port`. To ensure that HAProxy matches against the NodePort, use the annotation `ingress.appscode.com/use-node-port: "false"`. This is also considered the default value for this annotation. So, if you do not provide this annotation, it will be considered set to `false`.

 - Using the `NodePort` assigned to the HAProxy service. In this scenarios, external traffic directly hits the HAProxy NodePort service. So, HAProxy will see that incoming traffic is using Host like `domain:node-port`. To ensure that HAProxy matches against the NodePort, use the annotation `ingress.appscode.com/use-node-port: "true"` .
To ensure that HAProxy service ports stay fixed, define them in the Ingress YAML following: https://github.com/voyagermesh/voyager/blob/master/docs/guides/ingress/configuration/node-port.md

Below is an example Ingress with `ingress.appscode.com/use-node-port` annotation:

```yaml
$ kubectl apply -f test-ingress.yaml

apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/type: NodePort
    ingress.appscode.com/use-node-port: "true"
spec:
  rules:
  - host: voyager.appscode.test
    http:
      paths:
      - path: /foo
        backend:
          service:
            name: test-server
            port:
              number: 80
```

```bash
$ curl -v -H 'Host: voyager.appscode.test:30800' 192.168.99.100:30800/foo
*   Trying 192.168.99.100...
* Connected to 192.168.99.100 (192.168.99.100) port 30800 (#0)
> GET /foo HTTP/1.1
> Host: voyager.appscode.test:30800
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Wed, 14 Feb 2018 09:47:43 GMT
< Content-Type: text/plain
< Transfer-Encoding: chunked
< Server: echoserver
<


Hostname: test-server-68ddc845cd-x7dtv

Pod Information:
	-no pod information available-

Server values:
	server_version=nginx: 1.13.3 - lua: 10008

Request Information:
	client_address=172.17.0.6
	method=GET
	real path=/foo
	query=
	request_version=1.1
	request_uri=http://voyager.appscode.test:8080/foo

Request Headers:
	accept=*/*
	connection=close
	host=voyager.appscode.test:30800
	user-agent=curl/7.47.0
	x-forwarded-for=172.17.0.1

Request Body:
	-no body in request-

* Connection #0 to host 192.168.99.100 left intact
```