---
title: Configure Ingress Keep Source IP
menu:
  product_voyager_7.0.0-rc.0:
    identifier: keep-source-ip-configuration
    name: Keep Source IP
    parent: config-ingress
    weight: 10
product_name: voyager
menu_name: product_voyager_7.0.0-rc.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Keep Source IP

You can preserve client source IP by setting annotation `ingress.appscode.com/keep-source-ip` to `true`.

For `LoadBalancer` type ingresses, the actual configuration generated depends on the underlying cloud provider.

- `GCE, GKE, Azure, ACS`: Sets `ExternalTrafficPolicy` to `Local` for services used to expose HAProxy. See [here](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip).
- `AWS`: Enables [accept-proxy](accept-proxy.md) that enforces the use of the PROXY protocol over any connection accepted by any of the sockets declared on the same line.

For `NodePort` type ingresses, it sets `ExternalTrafficPolicy` to `Local` regardless the cloud provider.

## Ingress Example

```yaml
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
  annotations:
    ingress.appscode.com/keep-source-ip: "true"
spec:
  rules:
  - host: voyager.appscode.test
    http:
      paths:
      - path: /foo
        backend:
          serviceName: test-server
          servicePort: 80
```