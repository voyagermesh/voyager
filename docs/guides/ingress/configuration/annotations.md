---
menu:
  product_voyager_5.0.0-rc.8:
    name: Annotations
    parent: ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_5.0.0-rc.8
section_menu_id: guides
---

# Configuration Options
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

## Custom Annotations to LoadBalancer Service or Pods

If the LoadBalancer service and Pods needs to be set custom annotations, those can be
set via two ingress options.

`ingress.appscode.com/annotations-service`
Json encoded annotations map that will be applied to LoadBalancer service.

ie.
```
ingress.appscode.com/annotations-service = {"foo": "bar", "service-annotation": "set"}
```
This will add the `foo:bar` and `service-annotation:set` to the Service annotation.


`ingress.appscode.com/annotations-pod`
Json encoded annotations map that will be applied to LoadBalancer pods.

ie.
```
ingress.appscode.com/annotations-pod = {"foo": "bar", "pod-annotation": "set"}
```
This will add the `foo:bar` and `pod-annotation:set` to all the pods' annotation.


## Ingress Annotations Support
This file defines the list of supported annotations by voyager that are used in other ingress controllers.
voyager intent to ensure maximum amount compatibility between different implementations.

Voyager supports applying specified annotations in ingress or in backend service. If applied to ingress
configuration will be applied on all backend of the ingress, if applied to service configurations will only apply on those backends.

## Ingress Annotations
All following annotations are assumed to be prefixed with `ingress.kubernetes.io/`.
Voyager also supports some particular defines annotation, [which are described here](#voyager-annotations).

| Name | Details | Annotation applies to |
| --- |----| --- |
| **[TLS](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md#tls-related)** |
| `ssl-passthrough` | Pass TLS connections directly to backend; do not offload.   |  ingress |
| `hsts` | [Enable HSTS](hsts.md) | ingress |
| `hsts-max-age` | [Specifies the time (in seconds) the browser should connect to the server using the HTTPS connection.](hsts.md) | ingress|
| `hsts-preload` | [Enable HSTS preload](hsts.md) | ingress |
| `hsts-include-subdomains` | [HSTS rule applies to all of the site's sub domains](hsts.md) | ingress |
| **[Authentication](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md#authentication-related)** |
| `auth-type` | [Enable Basic Auth](basic-auth.md) | ingress, service |
| `auth-secret` | [Basic Auth user secret](basic-auth.md) | ingress, service |
| `auth-realm` | [Basic Auth realm](basic-auth.md) | ingress, service |
| **[Miscellaneous](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md#miscellaneous)** |
| `enable-cors` | [Enables CORS headers in HTTP response](cors.md) | ingress |
| `affinity` | [Sticky session](sticky-session.md). only supported value is cookie | ingress, service |
| `session-cookie-name` | [Sticky session cookie name to set](sticky-session.md) | ingress, service |
| `session-cookie-hash` | [Sticky session cookie type](sticky-session.md) | ingress, service |
| `proxy-body-size` | Maximum http request body size. This limits the advertised length of the HTTP request's body in bytes. | ingress |


## Voyager Annotations
voyager supports other annotations to serve different purpose prefixed with `ingress.appscode.com/`. Annotations
can be applied on ingress or backends.

| Name | Details | Annotation applies to |
|------|---------|---------------------|
| type | Defines loadbalancer type | ingress |
| replicas | [Scale load balancer replica](/docs/guides/ingress/replicas-and-autoscaling.md)| ingress |
| backend-weight | [Weighted Loadbalancing for Canary Deployment](weighted.md)| pod |
| annotations-service | [Add Custom Annotation to LoadBalancer Service](annotations.md)| ingress |
| annotations-pod | [Add Custom Annotation to LoadBalancer Pods](annotations.md) | ingress |
| accept-proxy | Accept proxy protocol | ingress |
| default-timeout | [Configure Custom Timeouts for HAProxy](configure-timeouts.md) | ingress |
| default-option | [Configure Options for HAProxy](configure-options.md) | ingress |
| backend-tls | [TLS enabled Backend](backend-tls.md) | service, ingress |
| sticky-session (deprecated) | [Configure Sticky session to Backends](sticky-session.md) | service, ingress |
| use-dns-resolver | [Supports redirects/DNS resolution for `ExternalName` type service](external-svc.md) | ingress |
| dns-resolver-nameservers | [Supports redirects/DNS resolution for `ExternalName` type service](external-svc.md) |ingress|
| dns-resolver-check-health | [Supports redirects/DNS resolution for `ExternalName` type service](external-svc.md) |ingress|
| dns-resolver-retries | [Supports redirects/DNS resolution for `ExternalName` type service](external-svc.md) | ingress|
| dns-resolver-timeout | [Supports redirects/DNS resolution for `ExternalName` type service](external-svc.md) |ingress|
| dns-resolver-hold | [Supports redirects/DNS resolution for `ExternalName` type service](external-svc.md)|ingress|
| stats | [Expose HAProxy stats](stats-and-prometheus.md) | ingress |
| stats-port | [Expose HAProxy stats](stats-and-prometheus.md) | ingress |
| stats-secret-name | [Expose HAProxy stats](stats-and-prometheus.md) | ingress |
| stats-service-name | [Expose HAProxy stats](stats-and-prometheus.md) | ingress |
| monitoring-agent | [Expose HAProxy stats using prometheus](stats-and-prometheus.md#using-prometheus) | ingress |
| service-monitor-labels |[Expose HAProxy stats using prometheus](stats-and-prometheus.md#using-prometheus) | ingress |
| service-monitor-namespace|[Expose HAProxy stats using prometheus](stats-and-prometheus.md#using-prometheus) | ingress |
| service-monitor-endpoint-port|[Expose HAProxy stats using prometheus](stats-and-prometheus.md#using-prometheus) | ingress |
| service-monitor-endpoint-scrape-interval |[Expose HAProxy stats using prometheus](stats-and-prometheus.md#using-prometheus) | ingress |


## Acknowledgements
 - [kubernetes/ingress](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md)
 - [Tracking Bugs](https://github.com/appscode/voyager/issues/491)
