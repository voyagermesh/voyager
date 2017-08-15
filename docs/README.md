[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/voyager)](https://goreportcard.com/report/github.com/appscode/voyager)

[Website](https://appscode.com) • [Slack](https://slack.appscode.com) • [Twitter](https://twitter.com/AppsCodeHQ)

# voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.


## Ingress
Voyager provides L7 and L4 loadbalancing using a custom Kubernetes [Ingress](docs/user-guide/ingress) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the options available in a standard Kubernetes [Ingress](https://kubernetes.io/docs/user-guide/ingress/). Here 
is a [complex ingress example](hack/example/ingress.yaml) that shows how various features can be used.
You can find the generated HAProxy Configuration [here](hack/example/haproxy_generated.cfg).

**Feautures**

  - [HTTP](docs/user-guide/ingress/single-service.md) and [TCP](docs/user-guide/ingress/tcp.md) loadbalancing,
  - [TLS Termination](docs/user-guide/ingress/tls.md),
  - Multi-cloud supports,
  - [Name and Path based virtual hosting](docs/user-guide/ingress/named-virtual-hosting.md),
  - [Cross namespace routing support](docs/user-guide/ingress/named-virtual-hosting.md#cross-namespace-traffic-routing),
  - [URL and Request Header Re-writing](docs/user-guide/ingress/header-rewrite.md),
  - [Wildcard Name based virtual hosting](docs/user-guide/ingress/named-virtual-hosting.md),
  - Persistent sessions, Loadbalancer stats.
  - [Route Traffic to StatefulSet Pods Based on Host Name](docs/user-guide/ingress/statefulset-pod.md)
  - [Weighted Loadbalancing for Canary Deployment](docs/user-guide/ingress/weighted.md)
  - [Customize generated HAProxy config via BackendRule](docs/user-guide/ingress/backend-rule.md) (can be used for [http rewriting](https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html), add [health checks](https://www.haproxy.com/doc/aloha/7.0/haproxy/healthchecks.html), etc.)
  - [Add Custom Annotation to LoadBalancer Service and Pods](docs/user-guide/ingress/annotations.md)
  - [Supports Loadbalancer Source Range](docs/user-guide/ingress/source-range.md)
  - [Supports redirects/DNS resolution for `ExternalName` type service](docs/user-guide/ingress/external-svc.md)
  - [Expose HAProxy stats for Prometheus](docs/user-guide/ingress/stats-and-prometheus.md)
  - [Supports AWS certificate manager](docs/user-guide/ingress/aws-cert-manager.md)
  - [Scale load balancer using HorizontalPodAutoscaling](docs/user-guide/ingress/replicas-and-autoscaling.md)
  - [Configure Custom Timeouts for HAProxy](docs/user-guide/ingress/configure-timeouts.md)

### Comparison with Kubernetes
| Feauture | [Kube Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) | AppsCode Ingress |
|----------|--------------|------------------|
| HTTP Loadbalancing| :white_check_mark: | :white_check_mark: |
| TCP Loadbalancing | :x: | :white_check_mark: |
| TLS Termination | :white_check_mark: | :white_check_mark: |
| Name and Path based virtual hosting | :x: | :white_check_mark: |
| Cross Namespace service support | :x: | :white_check_mark: |
| URL and Header rewriting | :x: | :white_check_mark: |
| Wildcard name virtual hosting | :x: | :white_check_mark: |
| Loadbalancer statistics | :x: | :white_check_mark: |
| Route Traffic to StatefulSet Pods Based on Host Name | :x: | :white_check_mark: |
| Weighted Loadbalancing for Canary Deployment| :x: | :white_check_mark: |
| Supports Loadbalancer Source Range | :x: | :white_check_mark: |
| Supports redirects/DNS resolve for `ExternalName` type service | :x: | :white_check_mark: |
| Expose HAProxy stats for Prometheus | :x: | :white_check_mark: |
| Supports AWS certificate manager | :x: | :white_check_mark: |

## Certificate
Voyager can automaticallty provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](docs/user-guide/certificate) resource. 

**Feautures**
- Provision free TLS certificates from Let's Encrypt,
- Manage issued certificates using a Kubernetes Third Party Resource,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.


### Supported Domain Providers
Read more about supported DNS Providers [here](/docs/user-guide/certificate/provider.md)

## Supported Versions
Kubernetes 1.3+


## User Guide
To deploy voyager in Kubernetes follow this [guide](docs/user-guide/README.md). In short this contains those two steps

1. Create `ingress.voyager.appscode.com` and `certificate.voyager.appscode.com` Third Party Resource
2. Deploy voyager to kubernetes.

## Running voyager alongside with other ingress controller
Voyager can be configured to handle default kubernetes ingress or only ingress.appscode.com. voyager can also be run
along side with other controllers.

```console
  --ingress-class
  // this flag can be set to 'voyager' to handle only ingress
  // with annotation kubernetes.io/ingress.class=voyager.

  // If unset, voyager will also handle ingress without ingress-class annotation.
```

## Developer Guide
Want to learn whats happening under the hood, read [the developer guide](docs/developer-guide/README.md).

## Contribution
If you're interested in being a contributor, read [the contribution guide](CONTRIBUTING.md).

## Building voyager
Read [Build Instructions](docs/developer-guide/build.md) to build voyager.

## Versioning Policy
There are 2 parts to versioning policy:
 - Operator version: Voyager __does not follow semver__, rather the _major_ version of operator points to the
Kubernetes [client-go](https://github.com/kubernetes/client-go#branches-and-tags) version. You can verify this
from the `glide.yaml` file. This means there might be breaking changes between point releases of the operator.
This generally manifests as changed annotation keys or their meaning.
Please always check the release notes for upgrade instructions.
 - TPR version: appscode.com/v1beta1 is considered in beta. This means any changes to the YAML format will be backward
compatible among different versions of the operator.

---

**The voyager operator collects anonymous usage statistics to help us learn how the software is being used and how we can improve it.
To disable stats collection, run the operator with the flag** `--analytics=false`.

---

## Acknowledgement
 - docker-library/haproxy https://github.com/docker-library/haproxy
 - kubernetes/contrib https://github.com/kubernetes/contrib/tree/master/service-loadbalancer
 - xenolf/lego https://github.com/appscode/lego
 - kelseyhightower/kube-cert-manager https://github.com/kelseyhightower/kube-cert-manager
 - PalmStoneGames/kube-cert-manager https://github.com/PalmStoneGames/kube-cert-manager
 - [Kubernetes cloudprovider implementation](https://github.com/kubernetes/kubernetes/tree/master/pkg/cloudprovider) 

## Support
If you have any questions, you can reach out to us.
* [Slack](https://slack.appscode.com)
* [Twitter](https://twitter.com/AppsCodeHQ)
* [Website](https://appscode.com)
