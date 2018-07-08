---
title: Using External Service as Ingress Backend | Kubernetes Ingress
menu:
  product_voyager_7.3.0:
    identifier: external-svc-backend-http
    name: External SVC
    parent: http-ingress
    weight: 40
product_name: voyager
menu_name: product_voyager_7.3.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Using External Service as Ingress Backend

You can use an external service as a Backend for Kubernetes Ingress. There are 2 options depending on whether the external service has an external IP or DNS record.

## Using External IP

You can introduce any [external IP address as a Kubernetes service](https://github.com/kubernetes/kubernetes/issues/8631#issuecomment-104404768) by creating a matching Service and Endpoint object. Then you can use this service as a backend for your Ingress rules.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-ip
spec:
  ports:
  - name: app
    port: 80
    protocol: TCP
    targetPort: 9855
  clusterIP: None
  type: ClusterIP
---
apiVersion: v1
kind: Endpoints
metadata:
  name: external-ip
subsets:
- addresses:
  # list all external ips for this service
  - ip: 172.17.0.5
  ports:
  - name: app
    port: 9855
    protocol: TCP
```

Now, you can use this `external-ip` Service as a backend in your Ingress definition. For example:

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ings-rhvulnlb
  namespace: test-x
spec:
  backend:
    serviceName: external-ip
    servicePort: "80"
```


## Using External Domain

You can introduce an external service into a Kubernetes cluster by creating a [`ExternalName`](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services---service-types) type service. Voyager can forward traffic in both HTTP and TCP mode to the named domain in the `ExternalName` service by resolving dns. For static resolution of DNS record when HAProxy config is parsed, only use the `ingress.appscode.com/use-dns-resolver: "true"` annotation on the respective service. To periodically resolve dns, [DNS resolvers](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#5.3) must be configured using annotations on the service name.

|  Keys  |   Value  |  Default |  Description |
|--------|-----------|----------|-------------|
| ingress.appscode.com/use-dns-resolver | bool | false for L7, always true for L4 | If set, DNS resolution will be used |
| ingress.appscode.com/dns-resolver-nameservers | array | | `Optional`. If set to an array of DNS nameservers, these will be used HAProxy to periodically resolve DNS. If not set, HAProxy parses the server line definition and matches a host name at start up. |
| ingress.appscode.com/dns-resolver-check-health | bool | | `Optional`. If nameservers are set, this defaults to `true`. Set to `false`, to disable periodic dns resolution. |
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

### HTTP Redirect
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

#### Default redirect
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

#### Backend Rule
If Backendrules are configured, Voyager will not auto generate any redirect rule. This allows users to use full spectrum of HTTP redirection options available in HAProxy. To learn about these option, consult [HAProxy documentation](https://www.haproxy.com/doc/aloha/7.0/haproxy/http_redirection.html#redirection-examples).

```
backend:
  backendRules:
  - http-request redirect location https://google.com code 302
  serviceName: external-svc-non-dns
  servicePort: "80"
```

