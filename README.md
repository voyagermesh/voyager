[Website](https://appscode.com) • [Slack](https://slack.appscode.com) • [Forum](https://discuss.appscode.com) • [Twitter](https://twitter.com/AppsCodeHQ)

# voyager
Voyager provides controller for [Ingress](#ingress) and [Certificates](#certificate) for Kubernetes developed by [AppsCode](https://appscode.com).


### Ingress
In here we call it ExtendedIngress.
An extended plugin of Kubernetes [Ingress](https://kubernetes.io/docs/user-guide/ingress/) by AppsCode, to support both L7 and L4 loadbalancing via a single ingress.
This is built on top of the [HAProxy](http://www.haproxy.org/), to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the features available in Kubernetes [Ingress](https://kubernetes.io/docs/user-guide/ingress/). Here 
is a [complex ingress example](hack/example/ingress.yaml) that shows how various features can be used.
You can find the generated HAProxy Configuration [here](hack/example/haproxy_generated.cfg).

**Feautures**

  - [HTTP](docs/user-guide/component/ingress/single-service.md) and [TCP](docs/user-guide/component/ingress/tcp.md) loadbalancing,
  - [TLS Termination](docs/user-guide/component/ingress/tls.md),
  - Multi-cloud supports,
  - [Name and Path based virtual hosting](docs/user-guide/component/ingress/named-virtual-hosting.md),
  - [Cross namespace routing support](docs/user-guide/component/ingress/named-virtual-hosting.md#cross-namespace-traffic-routing),
  - [URL and Request Header Re-writing](docs/user-guide/component/ingress/header-rewrite.md),
  - [Wildcard Name based virtual hosting](docs/user-guide/component/ingress/named-virtual-hosting.md),
  - Persistent sessions, Loadbalancer stats.


### Comparison with Kubernetes
| Feauture | Kube Ingress | AppsCode Ingress |
|----------|--------------|------------------|
| HTTP Loadbalancing| :white_check_mark: | :white_check_mark: |
| TCP Loadbalancing | :x: | :white_check_mark: |
| TLS Termination | :white_check_mark: | :white_check_mark: |
| Name and Path based virtual hosting | :x: | :white_check_mark: |
| Cross Namespace service support | :x: | :white_check_mark: |
| URL and Header rewriting | :x: | :white_check_mark: |
| Wildcard name virtual hosting | :x: | :white_check_mark: |
| Loadbalancer statistics | :x: | :white_check_mark: |


### Certificate
Kubernetes Controller to manage TLS Certificate.

**Feautures**
- Manage Kubernetes TLS secrets backed by Custom Certificate Provider, uses Let's Encrypt by default,
- Manage issued certificates based on Kubernetes ThirdParty Resources,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.


### Supported Domain Providers
Read more about supported DNS Providers [here](/docs/user-guide/component/certificate/provider.md)

## Supported Versions
Kubernetes 1.3+


## User Guide
To deploy voyager in Kubernetes follow this [guide](docs/user-guide/README.md). In short this contains those two steps

1. Create `ingress.appscode.com` and `certificate.appscode.com` Third Party Resource
2. Deploy voyager to kubernetes.

## Running voyager alongside with other ingress controller
Voyager can be configured to handle default kubernetes ingress or only ingress.appscode.com. voyager can also be run
along side with other controllers.

```sh
  --ingress-class
  // this flag can be set to 'voyager' to handle only ingress
  // with annotation kubernetes.io/ingress.class=voyager.

  // If unset, voyager will also handle ingress without ingress-class annotation.
```

## Developer Guide
Want to learn whats happening under the hood, read [the developer guide](docs/developer-guide/README.md).

## Contribution
If you're interested in being a contributor, read [the contribution guide](docs/contribution/README.md).


## Building voyager
Read [Build Instructions](docs/developer-guide/build.md) to build voyager.

## Acknowledgement
 - docker-library/haproxy https://github.com/docker-library/haproxy
 - kubernetes/contrib https://github.com/kubernetes/contrib/tree/master/service-loadbalancer
 - appscode/lego https://github.com/appscode/lego
 - kelseyhightower/kube-cert-manager https://github.com/kelseyhightower/kube-cert-manager
 - PalmStoneGames/kube-cert-manager https://github.com/PalmStoneGames/kube-cert-manager

## Support
If you have any questions, you can reach out to us.
* [Slack](https://slack.appscode.com)
* [Forum](https://discuss.appscode.com)
* [Twitter](https://twitter.com/AppsCodeHQ)
* [Website](https://appscode.com)
