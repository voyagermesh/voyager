[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/voyager)](https://goreportcard.com/report/github.com/appscode/voyager)

[Website](https://appscode.com) • [Slack](https://slack.appscode.com) • [Twitter](https://twitter.com/AppsCodeHQ)

# Voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.


## Ingress
Voyager provides L7 and L4 loadbalancing using a custom Kubernetes [Ingress](/docs/tasks/ingress) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the options available in a standard Kubernetes [Ingress](https://kubernetes.io/docs/tasks/ingress/).

## Certificate
Voyager can automaticallty provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](/docs/tasks/certificate) resource.

**Features**
- Provision free TLS certificates from Let's Encrypt,
- Manage issued certificates using a Kubernetes Third Party Resource,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.

## Supported Versions
Please pick a version of Voyager that matches your Kubernetes installation.

| Voyager Version                                                                        | Docs                                                                    | Kubernetes Version | Prometheus operator Version |
|----------------------------------------------------------------------------------------|-------------------------------------------------------------------------|--------------------|-----------------------------|
| [5.0.0-rc.7](https://github.com/appscode/voyager/releases/tag/5.0.0-rc.7) (uses CRD) | [User Guide](https://github.com/appscode/voyager/tree/5.0.0-rc.7/docs) | 1.7.x+             | 0.12.0+                     |
| [3.2.2](https://github.com/appscode/voyager/releases/tag/3.2.2) (uses TPR)             | [User Guide](https://github.com/appscode/voyager/tree/3.2.2/docs)       | 1.5.x - 1.7.x      | < 0.12.0                    |


## Installation
To install Voyager, please follow the guide [here](/docs/setup/install.md).

## Using Voyager
Want to learn how to use Voyager? Please start [here](/docs/README.md).

## Contribution guidelines
Want to help improve Voyager? Please start [here](/CONTRIBUTING.md).

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
 - kubernetes/ingress https://github.com/kubernetes/ingress
 - xenolf/lego https://github.com/appscode/lego
 - kelseyhightower/kube-cert-manager https://github.com/kelseyhightower/kube-cert-manager
 - PalmStoneGames/kube-cert-manager https://github.com/PalmStoneGames/kube-cert-manager
 - [Kubernetes cloudprovider implementation](https://github.com/kubernetes/kubernetes/tree/master/pkg/cloudprovider)

## Support
If you have any questions, you can reach out to us.
* [Slack](https://slack.appscode.com)
* [Twitter](https://twitter.com/AppsCodeHQ)
