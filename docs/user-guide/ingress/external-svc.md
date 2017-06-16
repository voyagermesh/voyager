Voyager supports `ExternalName` type services via dns resolution or http redirect.

## DNS Resolution
Voyager can forward traffic in both HTTP and TCP mode to the named domain in the external name
service by resolving dns. For static resolution of DNS address at the HAProxy config is parsed,
only use the `ingress.appscode.com/use-dns-resolver: "true"` on respective service. To periodically resolve
dns, [DNS resolvers](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3) must be configured using annotations on the service name.

|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/use-dns-resolver | bool | false | If set, DNS resolution will be used |
| ingress.appscode.com/dns-resolver-nameservers | array | | If set to an array of DNS nameservers, these will be used HAProxy to periodically resolve DNS. If not set, HAProxy parses the server line definition and matches a host name at start up. |
| ingress.appscode.com/dns-resolver-retries | integer | | If set, this defines the number of queries to send to resolve a server name before giving up. If not set, default value pre-configured by HAProxy is used. |
| ingress.appscode.com/dns-resolver-timeout | map | | If set, defines timeouts related to name resolution. Define value as '{ "event": "time" }'. For a list of valid events, please consult [HAProxy documentation](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3.2-timeout). |
| ingress.appscode.com/dns-resolver-hold | map | | If set, Defines period during which the last name resolution should be kept based on last resolution status. Define value as '{ "status": "period" }'. For a list of valid status, please consult [HAProxy documentation](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3.2-hold). |

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
apiVersion: appscode.com/v1beta1
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
If `ingress.appscode.com/use-dns-resolver` annotation is not set or missing on a `ExternalName` type service, Voyager resolves L7 ingress in the following ways.

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

### Backend Rule
```
backend:
  backendRule:
  - http-request redirect location https://google.com code 302
  serviceName: external-svc-non-dns
  servicePort: "80"
```
Manually setting custom redirect rules as backendRule will allow full customization to the redirection. You can also apply other customization before
http-request redirect.

### Default redirect
```
backend:
  serviceName: external-svc-non-dns
  servicePort: "80"
```
If both dns resolver and backendRule is missing then voyager will configure HAProxy to redirect traffic to provided domain and port.
The redirect code will be 301 (permanent redirect).

### TCP
If dns resolver is not set for an tcp service the generated HAProxy config will look like following

```
server server-name google.com:80
```

So HAProxy will get traffic from the domain and forward to client.
