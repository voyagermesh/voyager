---
menu:
  product_voyager_5.0.0-rc.8:
    name: Source Range
    parent: ingress
    weight: 110
product_name: voyager
menu_name: product_voyager_5.0.0-rc.8
section_menu_id: guides
---


## Loadbalancer Source Range
When using an Ingress with `ingress.appscode.com/type: LoadBalancer` annotation, you can specify the IP ranges
that are allowed to access the load balancer by using `spec.loadBalancerSourceRanges`.
This field takes a list of IP CIDR ranges, which will be forwarded to Kubernetes, that  will use to
configure firewall exceptions. This feature is currently supported on Google Compute Engine,
Google Container Engine and AWS. This field will be ignored if the cloud provider does not support the feature.

Assuming 10.0.0.0/8 is the internal subnet. In the following example, a load balancer will be created
that is only accessible to cluster internal ips. This will not allow clients from outside of your
Kubernetes cluster to access the load balancer.

```
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  loadBalancerSourceRanges:
  - 10.0.0.0/8
```

In the following example, a load balancer will be created that is only accessible to clients with
IP addresses from 130.211.204.1 and 130.211.204.2.
```
apiVersion: voyager.appscode.com/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: default
spec:
  rules:
  - host: appscode.example.com
    http:
      paths:
      - backend:
          serviceName: test-service
          servicePort: '80'
  loadBalancerSourceRanges:
  - 130.211.204.1/32
  - 130.211.204.2/32
```

NB: Currently there is a [bug in Kubernetes](https://github.com/kubernetes/kubernetes/issues/34218) due to which changing `loadBalancerSourceRanges` does not change SecurityGroup in AWS.
