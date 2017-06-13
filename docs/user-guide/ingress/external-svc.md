voyager supports `ServiceTypeExternalName` with redirect or by resolving DNS at runtime.

## Resolve DNS
voyager can forward traffic in both HTTP and TCP mode to the named domain in the external name
service via resolving the dns at runtime.

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

## Redirect Traffic
if dns resolver is not set in an external name service, voyager can redirect traffic via two ways.

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