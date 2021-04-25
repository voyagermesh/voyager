# Voyager

[![Go Report Card](https://goreportcard.com/badge/voyagermesh.dev/voyager)](https://goreportcard.com/report/voyagermesh.dev/voyager)
[![Build Status](https://github.com/voyagermesh/voyager/workflows/CI/badge.svg)](https://github.com/voyagermesh/voyager/actions?workflow=CI)
[![Docker Pulls](https://img.shields.io/docker/pulls/appscode/voyager.svg)](https://hub.docker.com/r/appscode/voyager/)
[![Slack](https://slack.appscode.com/badge.svg)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/voyagermesh.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=voyagermesh)

> Secure HAProxy Ingress Controller for Kubernetes

Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by [AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.

## Ingress
Voyager provides L7 and L4 loadbalancing using a custom Kubernetes [Ingress](https://voyagermesh.com/docs/latest/guides/ingress/) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the options available in a standard Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

## Certificate
Voyager can automatically provision and refresh SSL certificates (including wildcard certificates) issued from Let's Encrypt using a custom Kubernetes [Certificate](https://voyagermesh.com/docs/latest/guides/certificate/) resource.

## Installation
To install Voyager, please follow the guide [here](https://voyagermesh.com/docs/latest/setup/install/).

## Using Voyager
Want to learn how to use Voyager? Please start [here](https://voyagermesh.com/docs/latest/welcome/).

## Contribution guidelines
Want to help improve Voyager? Please start [here](https://voyagermesh.com/docs/latest/welcome/contributing/).

## Acknowledgement
 - docker-library/haproxy https://github.com/docker-library/haproxy
 - kubernetes/contrib https://github.com/kubernetes/contrib/tree/master/service-loadbalancer
 - kubernetes/ingress https://github.com/kubernetes/ingress
 - xenolf/lego https://github.com/appscode/lego
 - kelseyhightower/kube-cert-manager https://github.com/kelseyhightower/kube-cert-manager
 - PalmStoneGames/kube-cert-manager https://github.com/PalmStoneGames/kube-cert-manager
 - [Kubernetes cloudprovider implementation](https://github.com/kubernetes/kubernetes/tree/master/pkg/cloudprovider)
 - openshift/generic-admission-server https://github.com/openshift/generic-admission-server
 - TimWolla/haproxy-auth-request https://github.com/TimWolla/haproxy-auth-request

## Support

We use Slack for public discussions. To chit chat with us or the rest of the community, join us in the [AppsCode Slack team](https://appscode.slack.com/messages/C0XQFLGRM/details/) channel `#general`. To sign up, use our [Slack inviter](https://slack.appscode.com/).

If you have found a bug with Voyager or want to request for new features, please [file an issue](https://github.com/voyagermesh/voyager/issues/new).
