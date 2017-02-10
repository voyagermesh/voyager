## Ingress
In Kubernetes, Services and Pods have IPs which are only routable by cluster network. Inside clusters managed by AppsCode,
you can route your traffic, to your specific apps, via three ways:

### Services
`Kubernetes Service Types` allow you to specify what kind of service you want. Default and base type is the
ClusterIP, which exposes a service to connection, from inside the cluster.
NodePort and LoadBalancer are the two types exposing services to external traffic. [[...]](http://kubernetes.io/docs/user-guide/services/#publishing-services---service-types)

### Ingress
An Ingress is a collection of rules which allow inbound connections to reach the cluster services.
It can be configured to give services externally-reachable urls, load balance traffic, terminate SSL,
offer name-based virtual hosting, etc. Users can request ingress by POSTing the Ingress resource to API server.  [[...]](http://kubernetes.io/docs/user-guide/ingress/)

### Appscode Ingress
An extended plugin of Kubernetes Ingress by AppsCode, to support both L7 and L4 load balancing via a single ingress.
This is built on top of the HAProxy, to support high availability, sticky sessions, name and path-based virtual
hosting. This plugin also support configurable application ports with all the features available in Kubernetes Ingress. [[ ... ]](#what-is-appscode-ingress)

### Core features of AppsCode Ingress:
  - HTTP and TCP load balancing,
  - TLS Termination,
  - Multi-cloud supports,
  - Name and Path based virtual hosting,
  - Cross namespace routing support,
  - URL and Request Header Re-writing,
  - Wildcard Name based virtual hosting,
  - Persistent sessions, Loadbalancer stats.

## AppsCode Ingress Flow
Typically, services and pods have IPs only routable by the cluster network. All traffic that ends up at an
edge router is either dropped or forwarded elsewhere. An AppsCode Ingress is a collection of rules that allow
inbound connections to reach the app running in the cluster, and of course though it the applications are recongnized
via service the traffic will bypass service and go directly to pod.
AppsCode Ingress can also be configured to give services externally-reachable urls, load balance traffic,
terminate SSL, offer name based virtual hosting etc.

This resource Type is backed by an controller called voyager which monitors and manages the resources of AppsCode Ingress Kind.
Which is used for maintain and HAProxy backed loadbalancer to the cluster for open communications inside cluster
from internet via the loadbalancer.<br>
Even when a resource for AppsCode Ingress type is created, the controller will treat it as a new loadbalancer
request and will create a new loadbalancer, based on the configurations.


## Dive Into AppsCode Ingress
Multiple scenario can happen with loadbalancer. AppsCode Ingress intends to resolve all these scenario
for a high-availability loadbalancer, inside a kubernetes cluster.

### The Endpoints are like:

|  VERB   |                     ENDPOINT                                | ACTION | BODY
|---------|-------------------------------------------------------------|--------|-------
|  GET    | /apis/appscode.com/v1beta1/namespace/`ns`/ingresss          | LIST   | nil
|  GET    | /apis/appscode.com/v1beta1/namespace/`ns`/ingresss/`name`   | GET    | nil
|  POST   | /apis/appscode.com/v1beta1/namespace/`ns`/ingresss          | CREATE | JSON
|  PUT    | /apis/appscode.com/v1beta1/namespace/`ns`/ingresss/`name`   | UPDATE | JSON
|  DELETE | /apis/appscode.com/v1beta1/namespace/`ns`/ingresss/`name`   | DELETE | nil

### Single Service Ingress
There are existing Kubernetes concepts which allows you to expose a single service. However, you can do so
through an AppsCode Ingress as well, simply by specifying a default backend with no rules.

```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  backend:
    serviceName: test-server
    servicePort: '80'
```

This will create a load balancer forwarding all traffic to `test-server` service, unconditionally. The
loadbalancer ip can be found inside `Status` Field of the loadbalancer described response. **If there are other
rules defined in Ingress then the loadbalancer will forward traffic to the `test-server` when no other `rule` is
matched.

**As Example:**

```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  backend:
    serviceName: test-server
    servicePort: '80'
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'

```
**Default Backend**: An Ingress with no rules, like the one shown in the previous section, sends all
traffic to a single default backend. You can use the same technique to tell a loadbalancer
where to find your website’s 404 page, by specifying a set of rules and a default backend.
Traffic is routed to your default backend if none of the Hosts in your Ingress matches the Host in
request header, and/or none of the paths match url of request.

This Ingress will forward traffic to `test-service` if request comes from the host `appscode.example.com` only.
Other requests will be forwarded to default backend.

Default Backend also supports `headerRule` and `rewriteRule`.

### Simple Fanout
As described previously, pods within kubernetes have ips only visible on the cluster network. So, we need
something at the edge accepting ingress traffic and proxy-ing it to right endpoints. This component
is usually a highly available loadbalancer(s). An Ingress allows you to keep number of loadbalancers
down to a minimum, for example, a setup can be like:


```
foo.bar.com -> load balancer -> / foo    s1:80
                                / bar    s2:80
```

would require an Ingress such as:
```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - path: "/foo"
        backend:
          serviceName: s1
          servicePort: '80'
      - path: "/bar"
        backend:
          serviceName: s2
          servicePort: '80'
```
The Ingress controller will provision an implementation specific loadbalancer that satisfies the Ingress,
as long as the services (s1, s2) exist. When it has done so, you will see the address of the loadbalancer under
the Status of Ingress.


### Name based virtual hosting
Name-based virtual hosts use multiple host names for the same IP address.

```
foo.bar.com --|               |-> foo.bar.com s1:80
              | load balancer |
bar.foo.com --|               |-> bar.foo.com s2:80
```
The following Ingress tells the backing loadbalancer to route requests based on the [Host header](https://tools.ietf.org/html/rfc7230#section-5.4).

```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          serviceName: s1
          servicePort: '80'
  - host: bar.foo.com
    http:
      paths:
      - backend:
          serviceName: s2
          servicePort: '80'
```

> AppsCode Ingress also support **wildcard** Name based virtual hosting.
If the `host` field is set to `*.bar.com`, Ingress will forward traffic for any subdomain of `bar.com`.
so `foo.bar.com` or `test.bar.com` will forward traffic to the desired backends.

### Header and URL Rewriting
AppsCode Ingress support header and URL modification at the loadbalancer level. To ensure simplicity,
the header and rewrite rules follow the HAProxy syntax as it is.
To add some rewrite rules in a http rule, the syntax is:
```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
          headerRule:
          - X-Forwarded-Host %[base]
          rewriteRule:
          - "^([^\\ :]*)\\ /(.*)$ \\1\\ /testings/\\2"
```
The rules specified in `headerRule` will be applicable to the request header before going to the backend.
those rules will be added in the request header if the header is already not present in the request header.
In the example `X-Forwarded-Host` header is added to the request if it is not already there, `%[base]` indicates
the base URL the load balancer received the requests.

The rules specified in `rewriteRule` are used to modify the request url including the host. Current example
will add an `/testings` prefix in every request URI before forwarding it to backend.

### TCP LoadBalancing
TCP load balancing is one of the core features of AppsCode Ingress. AppsCode Ingress can handle
TCP Load balancing with or without TLS. One AppsCode Ingress can also be used to load balance both
HTTP and TCP together.

One Simple TCP Rule Would be:
```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - tcp:
    - host: appscode.example.com
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'

```

For this configuration, the loadbalancer will listen to `9899` port for incoming connections, and will
pass any request coming to it to the desired backend.

> For one Ingress Type you cannot have multiple rules listening to same port, even if they do not have
same `host`.
For TCP rules host parameters do not have much effective value.

## TLS
You can secure an Ingress by specifying a secret containing TLS pem or By referring a `certificate.appscode.com` resource.
Referring `certificate.appscode.com` will try to manage an certificate resource and use that certificate to encrypt communication.
We will discuss those things later.
Currently the Ingress only supports a
single TLS port, **443 for HTTP Rules**, and **Any Port for TCP Rules** and **assumes TLS termination**.

### HTTP TLS
For HTTP, If the TLS configuration section in an Ingress specifies different hosts, they will be multiplexed
on the same port according to hostname specified through SNI TLS extension
(Ingress controller supports SNI). The TLS secret must contain pem file to use for TLS, with a
key name ending with `.pem`. eg:

```
apiVersion: v1
kind: Secret
metadata:
  name: testsecret
  namespace: default
data:
  tls.crt: base64 encoded cert
  tls.key: base64 encoded key
```

Referencing this secret in an Ingress will tell the Ingress controller to secure the channel from
client to the loadbalancer using TLS:
```yaml
apiVersion: appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  tls:
  - secretName: testsecret
    hosts:
    - appscode.example.com
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'

```
This Ingress will open an `https` listener to secure the channel from the client to the loadbalancer,
terminate TLS at load balancer with the secret retried via SIN and forward unencrypted traffic to the
`test-service`.

### TCP TLS
Adding a TCP TLS termination at AppsCode Ingress is slightly different than HTTP, as TCP do not have
SNI advantage. An TCP endpoint with TLS termination, will look like this in AppsCode Ingress:
```yaml
apiVersion: appscode.com/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - tcp:
    - host: appscode.example.com
      port: '9898'
      backend:
        serviceName: tcp-service
        servicePort: '50077'
      secretName: testsecret

```
You need to set  the secretName field with the TCP rule to use a certificate.

### Global Configurations
AppsCode Ingress have some global configurations passed via the `annotaion` field of Ingress Metadata,
and those configuration will be applicable on load balancer globally. Annotation keys and its actions are as follows:

```
ingress.appscode.com/stickySession         = indicates the session affinity for the traffic, is set
                                      session affinity will apply to all the rulses set.
                                      defaults to false

ingress.appscode.com/type                  = indicates loadbalancer type to run via Kubernets Service
                                      Load balancer or in Node Port Mode.
                                      Values in:
                                         - LoadBalancer (default)
                                         - Daemon

ingress.appscode.com/daemon.hostname       = only applicatble when lb.appscode.com/type is set to Daemon,
                                      this hostname will indicate which host the load balancer
                                      needs to run.


ingress.appscode.com/ip                    = provide ip to run loadbalancer on the ip, it will only work
                                      if the ip is available to the clod provider. Works best with
                                      a persistance ip from the underlying cloud provider
                                      and set the ip as the value.

ingress.appscode.com/loadbalancer.persist  = if set to true load balancer will run in node port mode.


ingress.appscode.com/stats                 = if set to true it will open HAProxy stats in IP's 1936 port.
                                      defaults to false.

ingress.appscode.com/stats.user
ingress.appscode.com/stats.password        = if the stats is on the username and password, to authenticate
                                      with, for stats.



The following annotations can be applied in an Ingress if we want to manage Certificate with the
same ingress resource. Learn more by reading the certificate doc.
 certificate.appscode.com/enabled
 certificate.appscode.com/name
 certificate.appscode.com/provider
 certificate.appscode.com/email
 certificate.appscode.com/provider-secret
 certificate.appscode.com/user-secret
 certificate.appscode.com/server-url
```

## Other CURD Operations
Applying other operation like update, delete to AppsCode Ingress is regular kubernetes resource operation.
