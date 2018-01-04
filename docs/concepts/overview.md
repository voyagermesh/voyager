---
title: Overview | Voyager
menu:
  product_voyager_5.0.0-rc.11:
    identifier: overview-concepts
    name: Overview
    parent: concepts
    weight: 10
product_name: voyager
menu_name: product_voyager_5.0.0-rc.11
section_menu_id: concepts
---

# Voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.


## Ingress
Voyager provides L7 and L4 loadbalancing using a custom Kubernetes [Ingress](/docs/guides/ingress) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the options available in a standard Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

- HTTP
  - [Exposing Service via Ingress](/docs/guides/ingress/http/single-service.md)
  - [Virtual Hosting](/docs/guides/ingress/http/virtual-hosting.md)
  - [Supports Loadbalancer Source Range](/docs/guides/ingress/http/source-range.md)
  - [URL and Request Header Re-writing](/docs/guides/ingress/http/rewrite-rules.md)
  - [Enable CORS](/docs/guides/ingress/http/cors.md)
  - [Custom HTTP Port](/docs/guides/ingress/http/custom-http-port.md)
  - [Using External Service as Ingress Backend](/docs/guides/ingress/http/external-svc.md)
  - [HSTS](/docs/guides/ingress/http/hsts.md)
  - [Forward Traffic to StatefulSet Pods](/docs/guides/ingress/http/statefulset-pod.md)
  - [Configure Sticky session to Backends](/docs/guides/ingress/http/sticky-session.md)
  - [Blue Green Deployments using weighted Loadbalancing](/docs/guides/ingress/http/weighted.md)
- TLS/SSL
  - [TLS Termination](/docs/guides/ingress/tls/tls.md)
  - [Backend TLS](/docs/guides/ingress/tls/backend-tls.md)
  - [Supports AWS certificate manager](/docs/guides/ingress/tls/aws-cert-manager.md)
- TCP
  - [TCP LoadBalancing](/docs/guides/ingress/tcp/tcp.md)
- Configuration
  - [Customize generated HAProxy config via BackendRule](/docs/guides/ingress/configuration/backend-rule.md) (can be used for [http rewriting](https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html), add [health checks](https://www.haproxy.com/doc/aloha/7.0/haproxy/healthchecks.html), etc.)
  - [Apply Frontend Rules](/docs/guides/ingress/configuration/frontend-rule.md)
  - [Supported Annotations](/docs/guides/ingress/configuration/annotations.md)
  - [Specify NodePort](/docs/guides/ingress/configuration/node-port.md)
  - [Configure global options](/docs/guides/ingress/configuration/default-options.md)
  - [Configure Custom Timeouts for HAProxy](/docs/guides/ingress/configuration/default-timeouts.md)
  - [Using Custom HAProxy Templates](/docs/guides/ingress/configuration/custom-templates.md)
- Security
  - [Configure Basic Auth for HTTP Backends](/docs/guides/ingress/security/basic-auth.md)
  - [TLS Authentication](/docs/guides/ingress/security/tls-auth.md)
- Monitoring
  - [Exposing HAProxy Stats](/docs/guides/ingress/monitoring/stats.md)
- [Scaling Ingress](/docs/guides/ingress/scaling.md)
- [Placement of Ingress Pods](/docs/guides/ingress/pod-placement.md)


## Certificate

Voyager can automagically provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](/docs/guides/certificate) resource.

- Provision free TLS certificates from Let's Encrypt,
- Manage issued certificates using a Kubernetes Third Party Resource,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.
