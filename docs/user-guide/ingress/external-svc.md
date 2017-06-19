Voyager supports `ExternalName` type services via dns resolution or http redirect.

## DNS Resolution
Voyager can forward traffic in both HTTP and TCP mode to the named domain in the external name
service by resolving dns. For static resolution of DNS address at the HAProxy config is parsed,
only use the `ingress.appscode.com/use-dns-resolver: "true"` on respective service. To periodically resolve
dns, [DNS resolvers](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3) must be configured using annotations on the service name.

|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/use-dns-resolver | bool | false for L7, always true for L4 | If set, DNS resolution will be used |
| ingress.appscode.com/dns-resolver-nameservers | array | | `Optional`. If set to an array of DNS nameservers, these will be used HAProxy to periodically resolve DNS. If not set, HAProxy parses the server line definition and matches a host name at start up. |
| ingress.appscode.com/dns-resolver-retries | integer | | `Optional`. If set, this defines the number of queries to send to resolve a server name before giving up. If not set, default value pre-configured by HAProxy is used. |
| ingress.appscode.com/dns-resolver-timeout | map | | `Optional`. If set, defines timeouts related to name resolution. Define value as '{ "event": "time" }'. For a list of valid events, please consult [HAProxy documentation](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3.2-timeout). |
| ingress.appscode.com/dns-resolver-hold | map | | `Optional`. If set, Defines period during which the last name resolution should be kept based on last resolution status. Define value as '{ "status": "period" }'. For a list of valid status, please consult [HAProxy documentation](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3.2-hold). |

Following example illustrates the scenario.

```
apiVersion: v1
kind: Service
metadata:
  name: external-ns
  namespace: default
  annotations:
    ingress.appscode.com/use-dns-resolver: "true"
    ingress.appscode.com/dns-resolver-nameservers: '["8.8.8.8:53", "8.8.4.4:53"]'
spec:
  externalName: google.com
  type: ExternalName
```

If this service is used in ingress, the traffic will forward to google.com's address.

```
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ings-rhvulnlb
  namespace: test-x
spec:
  backend:
    serviceName: external-ns
    servicePort: "80"
```

## HTTP Redirect
If `ingress.appscode.com/use-dns-resolver` annotation is set to false or missing on a `ExternalName` type service,
Voyager resolves L7 ingress in the following ways.

```
apiVersion: v1
kind: Service
metadata:
  name: external-ns
  namespace: default
spec:
  externalName: google.com
  type: ExternalName
```

### Default redirect
If No BackendRules are configured for the endpoint, Voyager will configure HAProxy to redirect traffic to provided domain and port.
The redirect code will be 301 (permanent redirect). Scheme (http or https) used by endpoint is preserved on redirect.
```
backend:
  serviceName: external-svc-non-dns
  servicePort: "80"
```

The generated redirect line in HAProxy config:
```
http-request redirect location http[s]://{{e.ExternalName}}:{{ e.Port }} code 301
```

### Backend Rule
If Backendrules are configured, Voyager will not auto generate any redirect rule. This allows users to use full spectrum of HTTP redirection options available
in HAProxy. To learn about these option, consult [HAProxy documentation](https://www.haproxy.com/doc/aloha/7.0/haproxy/http_redirection.html#redirection-examples).

```
backend:
  backendRule:
  - http-request redirect location https://google.com code 302
  serviceName: external-svc-non-dns
  servicePort: "80"
```
