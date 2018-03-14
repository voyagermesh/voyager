[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/voyager)](https://goreportcard.com/report/github.com/appscode/voyager)
[![Build Status](https://travis-ci.org/appscode/voyager.svg?branch=master)](https://travis-ci.org/appscode/voyager)
[![codecov](https://codecov.io/gh/appscode/voyager/branch/master/graph/badge.svg)](https://codecov.io/gh/appscode/voyager)
[![Docker Pulls](https://img.shields.io/docker/pulls/appscode/voyager.svg)](https://hub.docker.com/r/appscode/voyager/)
[![Slack](https://slack.appscode.com/badge.svg)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/appscodehq.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=AppsCodeHQ)

# Voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.

## Ingress
Voyager provides L7 and L4 loadbalancing using a custom Kubernetes [Ingress](https://appscode.com/products/voyager/6.0.0/guides/ingress/) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the options available in a standard Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

## Certificate
Voyager can automaticallty provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](https://appscode.com/products/voyager/6.0.0/guides/certificate/) resource.

## Supported Versions
Please pick a version of Voyager that matches your Kubernetes installation.

| Voyager Version                                                                        | Docs                                                                    | Kubernetes Version | Prometheus operator Version |
|----------------------------------------------------------------------------------------|-------------------------------------------------------------------------|--------------------|-----------------------------|
| [6.0.0](https://github.com/appscode/voyager/releases/tag/6.0.0) (uses CRD)   | [User Guide](https://appscode.com/products/voyager/6.0.0/)         | 1.8.x+             | 0.16.0+                     |
| [5.0.0-rc.11](https://github.com/appscode/voyager/releases/tag/5.0.0-rc.11) (uses CRD) | [User Guide](https://appscode.com/products/voyager/5.0.0-rc.11/)        | 1.7.x              | 0.12.0+                     |
| [3.2.2](https://github.com/appscode/voyager/releases/tag/3.2.2) (uses TPR)             | [User Guide](https://github.com/appscode/voyager/tree/3.2.2/docs)       | 1.5.x - 1.7.x      | < 0.12.0                    |

## Installation
To install Voyager, please follow the guide [here](https://appscode.com/products/voyager/6.0.0/setup/install/).

## Using Voyager
Want to learn how to use Voyager? Please start [here](https://appscode.com/products/voyager/6.0.0/welcome/).

## Contribution guidelines
Want to help improve Voyager? Please start [here](https://appscode.com/products/voyager/6.0.0/welcome/contributing/).

---

**Voyager binaries collects anonymous usage statistics to help us learn how the software is being used and how we can improve it.
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
 - openshift/generic-admission-server https://github.com/openshift/generic-admission-server

## Support

We use Slack for public discussions. To chit chat with us or the rest of the community, join us in the [AppsCode Slack team](https://appscode.slack.com/messages/C0XQFLGRM/details/) channel `#general`. To sign up, use our [Slack inviter](https://slack.appscode.com/).

If you have found a bug with Voyager or want to request for new features, please [file an issue](https://github.com/appscode/voyager/issues/new).
