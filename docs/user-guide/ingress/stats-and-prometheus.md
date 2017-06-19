## Exposing HAProxy Stats
To expose HAProxy stats, please use the following annotations:

### Stats annotations
|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/stats | bool | false | `Optional`. If set, HAProxy stats will be exposed |
| ingress.appscode.com/stats-port | integer | 56789 | `Optional`. Port used to expose HAProxy stats |
| ingress.appscode.com/stats-secret-name | string | x | `Optional`. Secret used to provide username & password to secure HAProxy stats endpoint. Secret must contain keys `username` and `password` |

Please note that stats port is not exposed to the internet via the service running in front of HAProxy pods.

## Using Prometheus
Voyager operator exposes Prometheus ready metrics via the following endpoints on port `:56790`:

 - `/metrics`: Scrape this to monitor operator.
 - `/extensions/v1beta1/namespaces/:ns/ingresses/:name/pods/:ip/metrics` :  Scrape this endpoint to monitor HAProxy running for a Kubernetes ingress
 - `/voyager.appscode.com/v1beta1/namespaces/:ns/ingresses/:name/metrics`: Scrape this endpoint to monitor HAProxy running for an AppsCode extended ingress

To change the port, use `--address` flag on Voyager opreator.

## Using [CoreOS Prometheus Operator](https://coreos.com/operators/prometheus/docs/latest/)
Voyager operator can create [service monitors](https://coreos.com/operators/prometheus/docs/latest/user-guides/running-exporters.html#create-a-matching-servicemonitor) for HAProxy pods. If enabled, a side-car exporter pod is run with HAProxy to expose Prometheus ready metrics via the following endpoints on port `:56790`:

 - `/extensions/v1beta1/namespaces/:ns/ingresses/:name/pods/:ip/metrics` :  Scrape this endpoint to monitor HAProxy running for a Kubernetes ingress
 - `/voyager.appscode.com/v1beta1/namespaces/:ns/ingresses/:name/metrics`: Scrape this endpoint to monitor HAProxy running for an AppsCode extended ingress

To enable this feature, please use the following annotations:

|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/monitoring-agent | string | | `Required`. Indicates the monitoring agent used. Only valid value currently is 'coreos-prometheus-operator' |
| ingress.appscode.com/service-monitor-labels | map | | `Required`. Indicates labels applied to service monitor. |
| ingress.appscode.com/service-monitor-namespace| string | | `Required`. Indicates namespace where service monitors are created. This must be the same namespace of the Prometheus instance. |
| ingress.appscode.com/service-monitor-endpoint-target-port| integer | 56790 | `Optional`. Indicates the port used by exporter side-car to expose Prometheus metrics endpoint. If the default port 56790 is used to expose traffic, change it to an unused port. |
| ingress.appscode.com/service-monitor-endpoint-scrape-interval | string | | `Optional`. Indicates the srace interval for HAProxy exporter endpoint

__Known Limitations:__ If the HAProxy stats password is updated, exporter must be restarted to use the new credentials. This issue is tracked [here](https://github.com/appscode/voyager/issues/212).
