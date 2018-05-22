---
title: Configure Ingress Annotations
menu:
  product_voyager_7.0.0-rc.2:
    identifier: annotations-configuration
    name: Annotations
    parent: config-ingress
    weight: 1
product_name: voyager
menu_name: product_voyager_7.0.0-rc.2
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Configure ingress with annotations

Below is the full list of supported annotations:

|  Keys  |   Value   |  Default |
|--------|-----------|----------|
| [ingress.appscode.com/type](/docs/concepts/README.md) | LoadBalancer, HostPort, NodePort, Internal | `LoadBalancer` |
| [ingress.appscode.com/api-schema](/docs/concepts/overview.md) | {APIGroup}/{APIVersion} | `voyager.appscode.com/v1beta1` |
| [ingress.appscode.com/accept-proxy](/docs/guides/ingress/configuration/accept-proxy.md) | bool | `false` |
| [ingress.appscode.com/affinity](/docs/guides/ingress/http/sticky-session.md) | `cookie` | |
| [ingress.appscode.com/session-cookie-hash](/docs/guides/ingress/http/sticky-session.md) | string | |
| [ingress.appscode.com/session-cookie-name](/docs/guides/ingress/http/sticky-session.md) | string | `SERVERID` |
| [ingress.appscode.com/hsts](/docs/guides/ingress/http/hsts.md) | bool | `true` |
| [ingress.appscode.com/hsts-include-subdomains](/docs/guides/ingress/http/hsts.md) | bool | `false` |
| [ingress.appscode.com/hsts-max-age](/docs/guides/ingress/http/hsts.md) | string | `15768000` |
| [ingress.appscode.com/hsts-preload](/docs/guides/ingress/http/hsts.md) | bool | `false` |
| [ingress.appscode.com/use-node-port](/docs/concepts/ingress-types/nodeport.md) | bool | `false` |
| [ingress.appscode.com/enable-cors](/docs/guides/ingress/http/cors.md) | bool | `false` |
| [ingress.appscode.com/cors-allow-headers](/docs/guides/ingress/http/cors.md) | string | `DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization` |
| [ingress.appscode.com/cors-allow-methods](/docs/guides/ingress/http/cors.md) | string | `GET,PUT,POST,DELETE,PATCH,OPTIONS` |
| [ingress.appscode.com/cors-allow-origin](/docs/guides/ingress/http/cors.md) | string | `*` |
| [ingress.appscode.com/default-option](/docs/guides/ingress/configuration/default-options.md) | map | `{"http-server-close": "true", "dontlognull": "true"}` |
| [ingress.appscode.com/default-timeout](/docs/guides/ingress/configuration/default-timeouts.md) | map | `{"connect": "50s", "server": "50s", "client": "50s", "client-fin": "50s", "tunnel": "50s"}` |
| [ingress.appscode.com/auth-type](/docs/guides/ingress/security/basic-auth.md) | `basic` | |
| [ingress.appscode.com/auth-realm](/docs/guides/ingress/security/basic-auth.md) | string | |
| [ingress.appscode.com/auth-secret](/docs/guides/ingress/security/basic-auth.md) | string | |
| [ingress.appscode.com/auth-tls-error-page](/docs/guides/ingress/security/tls-auth.md) | string | |
| [ingress.appscode.com/auth-tls-secret](/docs/guides/ingress/security/tls-auth.md) | string | |
| [ingress.appscode.com/auth-tls-verify-client](/docs/guides/ingress/security/tls-auth.md) | `required` or, `optional` | `required` |
| [ingress.appscode.com/backend-tls](/docs/guides/ingress/tls/backend-tls.md) | string | |
| [ingress.appscode.com/replicas](/docs/guides/ingress/scaling.md) | int | `1` |
| [ingress.appscode.com/backend-weight](/docs/guides/ingress/http/blue-green-deployment.md) | int | |
| [ingress.appscode.com/whitelist-source-range](/docs/guides/ingress/configuration/whitelist.md) | string | |
| [ingress.appscode.com/max-connections](/docs/guides/ingress/configuration/max-connections.md) | int | |
| [ingress.appscode.com/ssl-redirect](/docs/guides/ingress/configuration/ssl-redirect.md) | bool | `true` |
| [ingress.appscode.com/force-ssl-redirect](/docs/guides/ingress/configuration/ssl-redirect.md) | bool | `false` |
| [ingress.appscode.com/limit-connection](/docs/guides/ingress/configuration/rate-limit.md) | int | |
| [ingress.appscode.com/limit-rpm](/docs/guides/ingress/configuration/rate-limit.md) | int | |
| [ingress.appscode.com/limit-rps](/docs/guides/ingress/configuration/rate-limit.md) | int | |
| [ingress.appscode.com/errorfiles](/docs/guides/ingress/configuration/error-files.md) | string | |
| [ingress.appscode.com/proxy-body-size](/docs/guides/ingress/configuration/body-size.md) | int | |
| [ingress.appscode.com/ssl-passthrough](/docs/guides/ingress/configuration/ssl-passthrough.md) | bool | `false` |
| [ingress.appscode.com/rewrite-target](/docs/guides/ingress/configuration/rewrite-target.md) | string | |
| [ingress.appscode.com/keep-source-ip](/docs/guides/ingress/configuration/keep-source-ip.md) | bool | `false` |
| [ingress.appscode.com/load-balancer-ip](/docs/guides/ingress/configuration/loadbalancer-ip.md) | string | |
| [ingress.appscode.com/annotations-pod](/docs/guides/ingress/configuration/pod-annotations.md) | map | |
| [ingress.appscode.com/annotations-service](/docs/guides/ingress/configuration/service-annotations.md) | map | |
| [ingress.appscode.com/stats](/docs/guides/ingress/monitoring/haproxy-stats.md) | bool | `false` |
| [ingress.appscode.com/stats-port](/docs/guides/ingress/monitoring/haproxy-stats.md) | int | `56789` |
| [ingress.appscode.com/stats-secret-name](/docs/guides/ingress/monitoring/haproxy-stats.md) | string | |
| [ingress.appscode.com/monitoring-agent](/docs/guides/ingress/monitoring/using-coreos-prometheus-operator.md) | string  |         |
| [ingress.appscode.com/service-monitor-labels](/docs/guides/ingress/monitoring/using-coreos-prometheus-operator.md) | map     |         |
| [ingress.appscode.com/service-monitor-namespace](/docs/guides/ingress/monitoring/using-coreos-prometheus-operator.md) | string  |         |
| [ingress.appscode.com/service-monitor-endpoint-port](/docs/guides/ingress/monitoring/using-coreos-prometheus-operator.md) | integer | 56790   |
| [ingress.appscode.com/service-monitor-endpoint-scrape-interval](/docs/guides/ingress/monitoring/using-coreos-prometheus-operator.md) | string  |         |
| [ingress.appscode.com/use-dns-resolver](/docs/guides/ingress/http/external-svc.md#using-external-domain) | bool | `false` |
| [ingress.appscode.com/dns-resolver-nameservers](/docs/guides/ingress/http/external-svc.md#using-external-domain) | string | |
| [ingress.appscode.com/dns-resolver-check-health](/docs/guides/ingress/http/external-svc.md#using-external-domain) | bool | `true` |
| [ingress.appscode.com/dns-resolver-retries](/docs/guides/ingress/http/external-svc.md#using-external-domain) | int | `0` |
| [ingress.appscode.com/dns-resolver-timeout](/docs/guides/ingress/http/external-svc.md#using-external-domain) | map | |
| [ingress.appscode.com/dns-resolver-hold](/docs/guides/ingress/http/external-svc.md#using-external-domain) | map | |
| [ingress.appscode.com/workload-kind](/docs/guides/ingress/pod-placement.md#choosing-workload-kind) | string | `Deployment` |
| [ingress.appscode.com/node-selector](/docs/guides/ingress/pod-placement.md#using-node-selector) | map | |
| [ingress.appscode.com/tolerations](/docs/guides/ingress/pod-placement.md#using-taints-and-toleration) | array | |
