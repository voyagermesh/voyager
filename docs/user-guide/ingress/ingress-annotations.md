---
menu:
  product_voyager_5.0.0-rc.5:
    name: Ingress Annotations
    parent: ingress
    weight: 80
product_name: voyager
left_menu: product_voyager_5.0.0-rc.5
section_menu_id: user-guide
---


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
| replicas | [Scale load balancer replica](/docs/user-guide/ingress/replicas-and-autoscaling.md)| ingress |
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