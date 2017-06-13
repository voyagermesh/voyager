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
  - Persistent sessions, Loadbalancer stats,
  - [Route Traffic to StatefulSet Pods Based on Host Name](statefulset-pod.md)
  - [Weighted Loadbalancing for Canary Deployment](weighted.md)
  - [Customize generated HAProxy config via BackendRule](backend-rule.md)
  - [Add Custom Annotation to LoadBalancer Service and Pods](annotations.md)

### Comparison with Kubernetes
| Feauture | Kube Ingress | AppsCode Ingress |
|----------|--------------|------------------|
| HTTP Loadbalancing| :white_check_mark: | :white_check_mark: |
| TCP Loadbalancing | :x: | :white_check_mark: |
| TLS Termination | :white_check_mark: | :white_check_mark: |
| Name and Path based virtual hosting | :x: | :white_check_mark: |
| Cross Namespace service support | :x: | :white_check_mark: |
| URL and Header rewriting | :x: | :white_check_mark: |
| Wildcard name virtual hosting | :x: | :white_check_mark: |
| Loadbalancer statistics | :x: | :white_check_mark: |
| Route Traffic to StatefulSet Pods Based on Host Name | :x: | :white_check_mark: |
| Weighted Loadbalancing on Canary Deployment| :x: | :white_check_mark: |
| Supports full Spectrum of HAProxy backend rules | :x: | :white_check_mark: |

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

## Ingress Status
If an ingress is created as `ingress.appscode.com/type: LoadBalancer` the ingress status field will contain
the ip/host name for that LoadBalancer. For `HostPort` mode the ingress will open traffic in nodes address the pod is running.

### Configurations Options
AppsCode Ingress have some global configurations passed via the `annotaions` field of Ingress Metadata,
and those configuration will be applicable on loadbalancer globally. Annotation keys and its actions are as follows:

|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|--------------|
| ingress.appscode.com/sticky-session | true, false | false | indicates the session affinity for the traffic, is set session affinity will apply to all the rulses set |
| ingress.appscode.com/type | LoadBalancer, HostPort, NodePOrt | LoadBalancer | indicates type of service used to expose HAproxy to the internet |
| ingress.appscode.com/replicas | integer | 1 | indicates number of replicas of HAProxy is run |
| ingress.appscode.com/node-selector | JSON | x | This nodeSelector will indicate which host the HAProxy is going to run. This is a required annotation for `HostPort` type ingress. The value of this annotation should be formatted as `{"foo": "bar", "foo2": "bar2"}`. This used to be called `ingress.appscode.com/daemon.nodeSelector` with comma seperated selectors list as `foo=bar,foo2=bar2`. This format is changed for the new key. We recommend you use the new key going forward. Any existing ingress with previous annotation will continue to function as expected. |
| ingress.appscode.com/ip | IP | x | This key is deprecated. Going forward, use `ingress.appscode.com/load-balaner-ip` |
| ingress.appscode.com/load-balaner-ip | IP | x | For "gce" and "gke" cloud provider, if this value is set to an valid IPv4 address, it will be assigned to Google cloud network loadbalancer used to expose HAProxy. Usually this is set to a static IP to preserve DNS configuration |
| ingress.appscode.com/annotations-service | JSON | x | Json encoded annotations to be applied in LoadBalancer Service |
| ingress.appscode.com/annotations-pod | JSON | x | Json encoded annotations to be applied in LoadBalancer Pods |
| ingress.appscode.com/stats | true, false | false | if set to true it will open HAProxy stats |
| ingress.appscode.com/stats-port | Integer | 1936 | HAProxy stats port to open via service |
| ingress.appscode.com/stats-secret-name | String | x | HAProxy stats secret name to use basic auth. Secret must contain key `username` `password` |
| ingress.appscode.com/stats-service-name | String | `stats-<ingress-name>` | Stats Service Name |


The following annotations can be applied in an Ingress if we want to manage Certificate with the
same ingress resource. Learn more by reading the [certificate doc](../certificate/README.md).
```
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
- [Virtual Hosting](named-virtual-hosting.md)
- [URL and Header Rewriting](header-rewrite.md)
- [TCP Loadbalancing](tcp.md)
- [TLS Termination](tls.md)
- [Route Traffic to StatefulSet Pods Based on Host Name](statefulset-pod.md)
- [Weighted Loadbalancing on Canary Deployment](weighted.md)
- [Supports full HAProxy Spectrum via BackendRule](backend-rule.md)

## Example
Check out examples for [complex ingress configurations](../../../../hack/example/ingress.yaml).
This example generates to a HAProxy Configuration like [this](../../../../hack/example/haproxy_generated.cfg).

## Other CURD Operations
Applying other operation like update, delete to AppsCode Ingress is regular kubernetes resource operation.
