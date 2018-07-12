---
title: CORS | Kubernetes Ingress
menu:
  product_voyager_7.4.0:
    identifier: cors-http
    name: CORS
    parent: http-ingress
    weight: 30
product_name: voyager
menu_name: product_voyager_7.4.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# CORS

Voyager can enable and configure CORS for all HTTP frontends via following ingress annotations:

- `ingress.appscode.com/enable-cors`: If set to `true` enables CORS for all HTTP Frontend. By default CORS is disabled.
- `ingress.appscode.com/cors-allow-headers`: Specifies allowed headers when CORS enabled. Default value is `DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization`.
- `ingress.appscode.com/cors-allow-methods`: Specifies allowed methods when CORS enabled. Default value is `GET,PUT,POST,DELETE,PATCH,OPTIONS`.
- `ingress.appscode.com/cors-allow-origin`: Specifies allowed origins when CORS enabled. Default value is `*`.

## Ingress Example

First create a test-server and expose it via service:

```console
$ kubectl run test-server --image=gcr.io/google_containers/echoserver:1.8
deployment "test-server" created

$ kubectl expose deployment test-server --type=LoadBalancer --port=80 --target-port=8080
service "test-server" exposed
```

Then create the ingress:

```yaml
$ kubectl apply -f test-ingress.yaml

apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/enable-cors: "true"
    ingress.appscode.com/cors-allow-headers: "Keep-Alive,User-Agent"
    ingress.appscode.com/cors-allow-methods: "GET,PUT"
    ingress.appscode.com/cors-allow-origin: "http://foo.example"
spec:
  rules:
  - host: voyager.appscode.test
    http:
      paths:
      - path: /foo
        backend:
          serviceName: test-server
          servicePort: 80
```

```console
$ kubectl get pods,svc
NAME                                       READY     STATUS    RESTARTS   AGE
po/test-server-68ddc845cd-x7dtv            1/1       Running   0          1d
po/voyager-test-ingress-5b758664f6-hjwjb   1/1       Running   0          20m

NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
svc/kubernetes             ClusterIP      10.96.0.1       <none>        443/TCP        3d
svc/test-server            LoadBalancer   10.105.13.31    <pending>     80:30390/TCP   1d
svc/voyager-test-ingress   LoadBalancer   10.106.53.141   <pending>     80:32218/TCP   1h

$ minikube service --url voyager-test-ingress
http://192.168.99.100:32218
```

Applying the annotation in ingress will have the following effects, will add the CORS Header in the response.

```console
$ curl -v -H 'Host: voyager.appscode.test' 192.168.99.100:32218/foo
*   Trying 192.168.99.100...
* Connected to 192.168.99.100 (192.168.99.100) port 32218 (#0)
> GET /foo HTTP/1.1
> Host: voyager.appscode.test
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 15 Feb 2018 05:06:49 GMT
< Content-Type: text/plain
< Transfer-Encoding: chunked
< Server: echoserver
< Access-Control-Allow-Origin: http://foo.example
< Access-Control-Allow-Methods: GET,PUT
< Access-Control-Allow-Credentials: true
< Access-Control-Allow-Headers: Keep-Alive,User-Agent
<


Hostname: test-server-68ddc845cd-x7dtv

Pod Information:
	-no pod information available-

Server values:
	server_version=nginx: 1.13.3 - lua: 10008

Request Information:
	client_address=172.17.0.5
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