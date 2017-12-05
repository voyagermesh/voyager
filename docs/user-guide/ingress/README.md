---
menu:
  product_voyager_5.0.0-rc.6:
    name: Overview
    parent: ingress
    weight: 8
product_name: voyager
left_menu: product_voyager_5.0.0-rc.6
section_menu_id: user-guide
url: /products/voyager/5.0.0-rc.6/user-guide/ingress/
aliases:
  - /products/voyager/5.0.0-rc.6/user-guide/ingress/README/
---

### Ingress
An Ingress is a collection of rules which allow inbound connections to reach the cluster services.
It can be configured to give services externally-reachable urls, load balance traffic, terminate SSL,
offer name-based virtual hosting, etc. Users can request ingress by POSTing the Ingress resource to API server.  [Read More](http://kubernetes.io/docs/user-guide/ingress/)

### Appscode Ingress
An extended plugin of Kubernetes Ingress by AppsCode, to support both L7 and L4 load balancing via a single ingress.
This is built on top of the HAProxy, to support high availability, sticky sessions, name and path-based virtual
hosting. This plugin also support configurable application ports with all the features available in Kubernetes Ingress. [Read More](#what-is-appscode-ingress)

**Features**
  - [HTTP](/docs/user-guide/ingress/single-service.md) and [TCP](/docs/user-guide/ingress/tcp.md) loadbalancing,
  - [TLS Termination](/docs/user-guide/ingress/tls.md),
  - Multi-cloud support,
  - [Name and Path based virtual hosting](/docs/user-guide/ingress/named-virtual-hosting.md),
  - [Cross namespace routing support](/docs/user-guide/ingress/named-virtual-hosting.md#cross-namespace-traffic-routing),
  - [URL and Request Header Re-writing](/docs/user-guide/ingress/header-rewrite.md),
  - [Wildcard Name based virtual hosting](/docs/user-guide/ingress/named-virtual-hosting.md),
  - Persistent sessions, Loadbalancer stats.
  - [Route Traffic to StatefulSet Pods Based on Host Name](/docs/user-guide/ingress/statefulset-pod.md)
  - [Weighted Loadbalancing for Canary Deployment](/docs/user-guide/ingress/weighted.md)
  - [Customize generated HAProxy config via BackendRule](/docs/user-guide/ingress/backend-rule.md) (can be used for [http rewriting](https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html), add [health checks](https://www.haproxy.com/doc/aloha/7.0/haproxy/healthchecks.html), etc.)
  - [Add Custom Annotation to LoadBalancer Service and Pods](/docs/user-guide/ingress/annotations.md)
  - [Supports Loadbalancer Source Range](/docs/user-guide/ingress/source-range.md)
  - [Supports redirects/DNS resolution for `ExternalName` type service](/docs/user-guide/ingress/external-svc.md)
  - [Expose HAProxy stats for Prometheus](/docs/user-guide/ingress/stats-and-prometheus.md)
  - [Supports AWS certificate manager](/docs/user-guide/ingress/aws-cert-manager.md)
  - [Scale load balancer using HorizontalPodAutoscaling](/docs/user-guide/ingress/replicas-and-autoscaling.md)
  - [Configure Custom Timeouts for HAProxy](/docs/user-guide/ingress/configure-timeouts.md)
  - [Custom port for HTTP](/docs/user-guide/ingress/custom-http-port.md)
  - [Specify NodePort](/docs/user-guide/ingress/node-port.md)
  - [Backend TLS](/docs/user-guide/ingress/backend-tls.md)
  - [Configure Options](/docs/user-guide/ingress/configure-options.md)
  - [Using Custom HAProxy Templates](/docs/user-guide/ingress/custom-templates.md)
  - [Configure Basic Auth for HTTP Backends](/docs/user-guide/ingress/basic-auth.md)
  - [Configure Sticky session to Backends](/docs/user-guide/ingress/sticky-session.md)
  - [Apply Frontend Rules](/docs/user-guide/ingress/frontend-rule.md)
  - [Supported Annotations](/docs/user-guide/ingress/ingress-annotations.md)

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
| Supports Loadbalancer Source Range | :x: | :white_check_mark: |
| Supports redirects/DNS resolve for `ExternalName` type service | :x: | :white_check_mark: |
| Expose HAProxy stats for Prometheus | :x: | :white_check_mark: |
| Supports AWS certificate manager | :x: | :white_check_mark: |

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
|  GET    | /apis/voyager.appscode.com/v1beta1/namespace/`ns`/ingresss          | LIST   | nil
|  GET    | /apis/voyager.appscode.com/v1beta1/namespace/`ns`/ingresss/`name`   | GET    | nil
|  POST   | /apis/voyager.appscode.com/v1beta1/namespace/`ns`/ingresss          | CREATE | JSON
|  PUT    | /apis/voyager.appscode.com/v1beta1/namespace/`ns`/ingresss/`name`   | UPDATE | JSON
|  DELETE | /apis/voyager.appscode.com/v1beta1/namespace/`ns`/ingresss/`name`   | DELETE | nil

## Ingress Status
If an ingress is created as `ingress.appscode.com/type: LoadBalancer` the ingress status field will contain
the ip/host name for that LoadBalancer. For `HostPort` mode the ingress will open ports on the nodes selected to run HAProxy.

### Configuration Options
Voyager operator allows customization of Ingress resource using annotation keys with `ingress.appscode.com/` prefix.
The ingress annotaiton keys are always string. Annotation values might have the following data types:

| Value Type | Description | Example YAML |
|----------- |-------------|--------------|
| string | any valid string | 'v1'; "v2"  |
| integer | any valid integer | '1'; "2" |
| bool | 1, t, T, TRUE, true, True considered _true_; everything else is considered _false_ | 'true' |
| array | json formatted array of string | '["v1", "v2"]' |
| map | json formatted string to string map | '{ "k1" : "v1", "k2": "v2" }' |
| enum | string which has a predefined set of valid values | 'E1'; "E2"  |

If you are using YAML to write your Ingress, you can use any valid YAML syntax, including multi-line string. Here is an example:
```yaml
annotations:
  ingress.appscode.com/type: LoadBalancer
  ingress.appscode.com/replicas: '2'
  ingress.appscode.com/load-balancer-ip: '100.101.102.103'
  ingress.appscode.com/stats: 'true'
  ingress.appscode.com/stats-port: '2017'
  ingress.appscode.com/stats-secret-name: my-secret
  ingress.appscode.com/annotations-service: |
    {
        "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "http",
        "service.beta.kubernetes.io/aws-load-balancer-proxy-protocol": "*",
        "service.beta.kubernetes.io/aws-load-balancer-ssl-cert": "arn:aws:acm:..."
    }
```

Below is the full list of supported annotation keys, [voyager also support standard ingress annotations](ingress-annotations.md):

|  Keys  |   Value   |  Default |  Description |
|--------|-----------|----------|--------------|
| ingress.appscode.com/type | LoadBalancer, HostPort, NodePort, Internal | LoadBalancer | `Required`. Indicates type of service used to expose HAProxy to the internet |
| ingress.appscode.com/replicas | integer | 1 | `Optional`. Indicates number of replicas of HAProxy pods |
| ingress.appscode.com/load-balancer-ip | string | x | `Optional`. For "gce", "gke", "azure", "acs" cloud provider, if this value is set to a valid IPv4 address, it will be assigned to loadbalancer used to expose HAProxy. The IP should be pre-allocated in cloud provider account but not assigned to the load-balancer. Usually this is set to a static IP to preserve DNS configuration |
| ingress.appscode.com/node-selector | map | x | Indicates which hosts are selected to run HAProxy pods. This is a recommended annotation for `HostPort` type ingress. |
| ingress.appscode.com/sticky-session | bool | false | `Optional`. Indicates the session affinity for the traffic. If set, session affinity will apply to all the rulses. |
| ingress.appscode.com/annotations-service | map | x | `Optional`. Annotaiotns applied to service used to expose HAProxy |
| ingress.appscode.com/annotations-pod | map | x | `Optional`. Annotations applied to pods used to run HAProxy |
| ingress.appscode.com/keep-source-ip | bool | false | `Optional`. If set, preserves source IP for `LoadBalancer` type ingresses. The actual configuration generated depends on the underlying cloud provider. For gce, gke, azure: Adds annotation `service.beta.kubernetes.io/external-traffic: OnlyLocal` to services used to expose HAProxy. For aws, enforces the use of the PROXY protocol. |
| ingress.appscode.com/accept-proxy | bool | false | `Optional`. If set, enforces the use of the PROXY protocol. |
| ingress.appscode.com/stats | bool | false | `Optional`. If set, HAProxy stats will be exposed |
| ingress.appscode.com/stats-port | integer | 56789 | `Optional`. Port used to expose HAProxy stats |
| ingress.appscode.com/stats-secret-name | string | x | `Optional`. Secret used to provide username & password to secure HAProxy stats endpoint. Secret must contain keys `username` and `password` |
| ingress.appscode.com/stats-service-name | string | `voyager-<ingress-name>-stats` | ClusterIP type service used to expose HAproxy stats. This allows to avoid exposing stats to internet. |
| ingress.appscode.com/ip | | | Removed since 1.5.6. Use `ingress.appscode.com/load-balancer-ip` |
| ingress.appscode.com/persist | | | Removed since 1.5.6. |
| ingress.appscode.com/daemon.nodeSelector | | | Removed since 1.5.6. Use `ingress.appscode.com/node-selector` |
| ingress.appscode.com/stickySession | | | Removed since 1.5.6. Use `ingress.appscode.com/sticky-session` |
| ingress.appscode.com/annotationsService | | | Removed since 1.5.6. Use `ingress.appscode.com/annotations-service` |
| ingress.appscode.com/annotationsPod | | | Removed since 1.5.6. Use `ingress.appscode.com/annotations-pod` |
| ingress.appscode.com/statsSecretName | | | Removed since 1.5.6. Use `ingress.appscode.com/stats-secret-name` |

**Following annotations for ingress are not modifiable. The configuration is applied only when an Ingress object is created.
If you need to update these annotations, then first delete the Ingress and then recreate.**
```
ingress.appscode.com/type
ingress.appscode.com/node-selector
ingress.appscode.com/load-balaner-ip
```
The issue is being [tracked here.](https://github.com/appscode/voyager/issues/143)

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
- [Add Custom Annotation to LoadBalancer Service and Pods](annotations.md)
- [Supports Loadbalancer Source Range](source-range.md)
- [Supports redirects/DNS resolve for `ServiceTypeExternalName`](external-svc.md)
- [Expose HAProxy stats and metrics, use prometheus with metrics](stats-and-prometheus.md)

## Example
Check out examples for [complex ingress configurations](../../../hack/example/ingress.yaml).
This example generates to a HAProxy Configuration like [this](../../../hack/example/haproxy_generated.cfg).

## Other CURD Operations
Applying other operation like update, delete to AppsCode Ingress is regular kubernetes resource operation.
