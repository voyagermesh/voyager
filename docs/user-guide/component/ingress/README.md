## Ingress
In Kubernetes, Services and Pods have IPs which are only routable by cluster network. Inside clusters managed by AppsCode,
you can route your traffic, to your specific apps, via three ways:

### Services
`Kubernetes Service Types` allow you to specify what kind of service you want. Default and base type is the
ClusterIP, which exposes a service to connection, from inside the cluster.
NodePort and LoadBalancer are the two types exposing services to external traffic. [Read More](http://kubernetes.io/docs/user-guide/services/#publishing-services---service-types)

### Ingress
An Ingress is a collection of rules which allow inbound connections to reach the cluster services.
It can be configured to give services externally-reachable urls, load balance traffic, terminate SSL,
offer name-based virtual hosting, etc. Users can request ingress by POSTing the Ingress resource to API server.  [Read More](http://kubernetes.io/docs/user-guide/ingress/)

### Appscode Ingress
An extended plugin of Kubernetes Ingress by AppsCode, to support both L7 and L4 load balancing via a single ingress.
This is built on top of the HAProxy, to support high availability, sticky sessions, name and path-based virtual
hosting. This plugin also support configurable application ports with all the features available in Kubernetes Ingress. [Read More](#what-is-appscode-ingress)

### Core features of AppsCode Ingress:
  - [HTTP](single-service.md) and [TCP](tcp.md) loadbalancing,
  - [TLS Termination](tls.md),
  - Multi-cloud supports,
  - [Name and Path based virtual hosting](named-virtual-hosting.md),
  - [Cross namespace routing support](named-virtual-hosting.md),
  - [URL and Request Header Re-writing](header-rewrite.md),
  - [Wildcard Name based virtual hosting](named-virtual-hosting.md),
  - Persistent sessions, Loadbalancer stats.

### Comparison with Kubernetes
| Feauture | Kube Ingress | AppsCode Ingress |
|----------|--------------|------------------|
| HTTP Loadbalancing| :white_check_mark: | :white_check_mark: |
| TCP Loadbalincing | :x: | :white_check_mark: |
| TLS Termination | :white_check_mark: | :white_check_mark: |
| Name and Path based virtual hosting | :x: | :white_check_mark: |
| Cross Namespace service support | :x: | :white_check_mark: |
| URL and Header rewriting | :x: | :white_check_mark: |
| Wildcard name virtual hosting | :x: | :white_check_mark: |
| Loadbalncer statistics | :x: | :white_check_mark: |

## AppsCode Ingress Flow
Typically, services and pods have IPs only routable by the cluster network. All traffic that ends up at an
edge router is either dropped or forwarded elsewhere. An AppsCode Ingress is a collection of rules that allow
inbound connections to reach the app running in the cluster, and of course though it the applications are recongnized
via service the traffic will bypass service and go directly to pod.
AppsCode Ingress can also be configured to give services externally-reachable urls, load balance traffic,
terminate SSL, offer name based virtual hosting etc.

This resource Type is backed by an controller called Voyager which monitors and manages the resources of AppsCode Ingress Kind.
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


### Configurations Options
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

ingress.appscode.com/daemon.nodeSelector       = only applicatble when lb.appscode.com/type is set to Daemon,
                                      this nodeSelector will indicate which host the load balancer
                                      needs to run.
                                      The format of providing nodeSelector is -
                                      `foo=bar,foo2=bar2`


ingress.appscode.com/ip                    = provide ip to run loadbalancer on the ip, it will only work
                                      if the ip is available to the clod provider. Works best with
                                      a persistance ip from the underlying cloud provider
                                      and set the ip as the value.

ingress.appscode.com/loadbalancer.persist  = if set to true load balancer will run in node port mode.


ingress.appscode.com/stats                 = if set to true it will open HAProxy stats in IP's 1936 port.
                                      defaults to false.

ingress.appscode.com/stats.secretName      = if the stats is on then this kubernetes secret will
                                      be used as stats basic auth. This secret must contain two data `username`
                                      and `password` which will be used.



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

## Next Reading
- [Single Service example](single-service.md)
- [Simple Fanout](simple-fanout.md)
- [Virtual Hosting](named-virtual-hostin.md)
- [URL and Header Rewriting](header-rewrite.md)
- [TCP Loadbalancing](tcp.md)
- [TLS Termination](tls.md)


## Example
Check out examples for [complex ingress configurations](../../../../hack/example/ingress.yaml).
This example generates to a HAProxy Configuration like [this](../../../../hack/example/haproxy_generated.cfg).

## Other CURD Operations
Applying other operation like update, delete to AppsCode Ingress is regular kubernetes resource operation.
