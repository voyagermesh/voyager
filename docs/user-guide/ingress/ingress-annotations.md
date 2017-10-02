## Ingress Annotations Support
This file defines the list of supported annotations by voyager that are used in other ingress controllers.
voyager intent to ensure maximum amount compatibility between different implementations.

Voyager supports applying specified annotations in ingress or in backend service. If applied to ingress
configuration will be applied on all backend of the ingress, if applied to service configurations will only apply on those backends.

## Ingress Annotations
All following annotations are assumed to be prefixed with `ingress.kubernetes.io/`. Voyager also supports some particular defines
annotation, [which are described here](#voyager-annotations).

| Name | Details | Annotation applies to |
| --- | :---: | --- |
| **[TLS](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md#tls-related)** |
| `ssl-passthrough` | Pass TLS connections directly to backend; do not offload.   |  ingress |
| `hsts` | :heavy_check_mark: |`FrontendRule` |
| `hsts-max-age` | :heavy_check_mark: | `FrontendRule`|
| `hsts-preload` | :heavy_check_mark: |`FrontendRule` |
| `hsts-include-subdomains` | :heavy_check_mark: | `FrontendRule`|
| **[Authentication](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md#authentication-related)** |
| `auth-type` | :heavy_check_mark: | [Basic Auth](https://github.com/appscode/voyager/blob/release-4.0/docs/user-guide/ingress/basic-auth.md) |
| `auth-secret` | :heavy_check_mark: | [Basic Auth](https://github.com/appscode/voyager/blob/release-4.0/docs/user-guide/ingress/basic-auth.md) |
| `auth-realm` | :heavy_check_mark: | [Basic Auth](https://github.com/appscode/voyager/blob/release-4.0/docs/user-guide/ingress/basic-auth.md) |
| **[Miscellaneous](https://github.com/kubernetes/ingress/blob/master/docs/annotations.md#miscellaneous)** |
| `enable-cors` | :heavy_check_mark:  | [Frontend Rule](http://blog.nasrulhazim.com/2017/07/haproxy-setting-up-cors/) |
| `affinity` | :heavy_check_mark: | [Sticky session](https://github.com/appscode/voyager/blob/release-4.0/docs/user-guide/ingress/sticky-session.md) |
| `session-cookie-name` | :heavy_check_mark: | [Custom template](https://github.com/appscode/voyager/blob/release-4.0/docs/user-guide/ingress/custom-templates.md) |
| `session-cookie-hash` | :heavy_check_mark: | |
| `proxy-body-size` | :heavy_check_mark:  ||


## Voyager Annotations
voyager supports other annotations to serve different purpose prefixed with `ingress.appscode.com/`. Annotations
can be applied on ingress or backends.

| Name | Details | Annotation applies to |
|------|---------|---------------------|
| type | Defines loadbalancer type | ingress |
| replicas | [Scale load balancer](/docs/user-guide/ingress/replicas-and-autoscaling.md)| ingress |
| backend-weight | [Weighted Loadbalancing for Canary Deployment](weighted.md)| pod |
| annotations-service | [Add Custom Annotation to LoadBalancer Service and Pods](annotations.md)| ingress |
| annotations-pod | [Add Custom Annotation to LoadBalancer Service and Pods](annotations.md) | ingress |
| accept-proxy | Accept proxy protocol | ingress |
| default-timeout | [Configure Custom Timeouts for HAProxy](configure-timeouts.md) | ingress |
| default-option | [Configure Options](configure-options.md) | ingress |
| backend-tls | [Backend TLS](backend-tls.md) | service, ingress |
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