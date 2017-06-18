## Exposing HAProxy Stats
To expose HAProxy stats, please use the following annotations: 

### Stats annotations
|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/stats | bool | false | If set, HAProxy stats will be exposed |
| ingress.appscode.com/stats-port | integer | 1936 | Port used to expose HAProxy stats |
| ingress.appscode.com/stats-secret-name | string | x | Secret used to provide username & password to secure HAProxy stats endpoint. Secret must contain keys `username` and `password` |
| ingress.appscode.com/stats-service-name | string | `<ingress-name>-stats` | ClusterIP type service used to expose HAproxy stats. This allows to avoid exposing stats to internet. |

Please note that a separate `ClusterIP` type service is used to expose stats. So, you can use expose unauthenticated stats endpoint without exposing them to the internet.

## Using Prometheus
Voyager operator exposes Prometheus ready metrics via the following endpoints on port `:8080`:

 - `/metrics`: Scrape this to monitor operator.
 - `/extensions/v1beta1/namespaces/:ns/ingresses/:name/pods/:ip/metrics` :  Scrape this endpoint to monitor HAProxy running for a Kubernetes ingress
 - `/voyager.appscode.com/v1beta1/namespaces/:ns/ingresses/:name/pods/:ip/metrics`: Scrape this endpoint to monitor HAProxy running for an AppsCode extended ingress

To change the port, use `--address` flag on Voyager opreator.

Currently [further discussion is on-going](https://github.com/appscode/voyager/issues/154) on how to integrate this with CoreOS Prometheus Operator.
