## Exposing HAProxy Stats
One can simply enable HAProxy stats by simply adding an annotation `ingress.appscode.com/stats: true`.
This will create a separate service to expose the HAProxy stats.

### Stats options
|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/stats | true, false | false | if set to true it will open HAProxy stats |
| ingress.appscode.com/stats-port | Integer | 1936 | HAProxy stats port to open via service |
| ingress.appscode.com/stats-secret-name | String | x | HAProxy stats secret name to use basic auth. Secret must contain key `username` `password` |
| ingress.appscode.com/stats-service-name | String | `stats-<ingress-name>` | Stats Service Name |


## Expose Prometheus Metrics
(TODO @tamal describe details)
This will allow monitoring operator & HAProxy pods using Prometheus. The endpoints are:

/metrics: Scrape this to monitor operator.
/extensions/v1beta1/namespaces/:ns/ingresses/:name/pods/:ip/metrics : Scrape this endpoint to monitor HAProxy running for a Kubernetes ingress
/appscode.com/v1beta1/namespaces/:ns/ingresses/:name/pods/:ip/metrics: Scrape this endpoint to monitor HAProxy running for an AppsCode extended ingress