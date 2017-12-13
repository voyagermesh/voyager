---
title: Overview | Voyager
description: Overview of Voyager
menu:
  product_voyager_5.0.0-rc.7:
    identifier: overview-voyager
    name: Overview
    parent: getting-started
    weight: 20
product_name: voyager
menu_name: product_voyager_5.0.0-rc.7
section_menu_id: getting-started
url: /products/voyager/5.0.0-rc.7/getting-started/
aliases:
  - /products/voyager/5.0.0-rc.7/
  - /products/voyager/5.0.0-rc.7/README/
---

# Voyager
Voyager is a [HAProxy](http://www.haproxy.org/) backed [secure](#certificate) L7 and L4 [ingress](#ingress) controller for Kubernetes developed by
[AppsCode](https://appscode.com). This can be used with any Kubernetes cloud providers including aws, gce, gke, azure, acs. This can also be used with bare metal Kubernetes clusters.


## Ingress
Voyager provides L7 and L4 loadbalancing using a custom Kubernetes [Ingress](/docs/tasks/ingress) resource. This is built on top of the [HAProxy](http://www.haproxy.org/) to support high availability, sticky sessions, name and path-based virtual hosting.
This also support configurable application ports with all the options available in a standard Kubernetes [Ingress](https://kubernetes.io/docs/tasks/ingress/).

**Features**
- HTTP
  - [Single Service Ingress](/docs/tasks/ingress/http/single-service.md)
  - [Name and Path based virtual hosting](/docs/tasks/ingress/http/named-virtual-hosting.md)
  - [Supports Loadbalancer Source Range](/docs/tasks/ingress/http/source-range.md)
  - [URL and Request Header Re-writing](/docs/tasks/ingress/http/header-rewrite.md)
  - [Enable CORS](/docs/tasks/ingress/http/cors.md)
  - [Custom HTTP Port](/docs/tasks/ingress/http/custom-http-port.md)
  - [Supports redirects/DNS resolution for `ExternalName` type service](/docs/tasks/ingress/http/external-svc.md)
  - [HSTS](/docs/tasks/ingress/http/hsts.md)
  - [Simple Fanout](/docs/tasks/ingress/http/simple-fanout.md)
  - [Route Traffic to StatefulSet Pods Based on Host Name](/docs/tasks/ingress/http/statefulset-pod.md)
  - [Configure Sticky session to Backends](/docs/tasks/ingress/http/sticky-session.md)
  - [Weighted Loadbalancing for Canary Deployment](/docs/tasks/ingress/http/weighted.md)
- TLS/SSL
  - [TLS Termination](/docs/tasks/ingress/tls/tls.md)
  - [Backend TLS](/docs/tasks/ingress/tls/backend-tls.md)
  - [Supports AWS certificate manager](/docs/tasks/ingress/tls/aws-cert-manager.md)
- TCP
  - [TCP LoadBalancing](/docs/tasks/ingress/tcp/tcp.md)
- Configuration
  - [Customize generated HAProxy config via BackendRule](/docs/tasks/ingress/configuration/backend-rule.md) (can be used for [http rewriting](https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html), add [health checks](https://www.haproxy.com/doc/aloha/7.0/haproxy/healthchecks.html), etc.)
  - [Apply Frontend Rules](/docs/tasks/ingress/configuration/frontend-rule.md)
  - [Supported Annotations](/docs/tasks/ingress/configuration/annotations.md)
  - [Bind to address](/docs/tasks/ingress/configuration/bind-address.md)
  - [Specify NodePort](/docs/tasks/ingress/configuration/node-port.md)
  - [Configure global options](/docs/tasks/ingress/configuration/configure-options.md)
  - [Configure Custom Timeouts for HAProxy](/docs/tasks/ingress/configuration/configure-timeouts.md)
  - [Using Custom HAProxy Templates](/docs/tasks/ingress/configuration/custom-templates.md)
- External DNS
  - [Configuring DNS](/docs/tasks/ingress/dns/external-dns.md)
- Security
  - [Configure Basic Auth for HTTP Backends](/docs/tasks/ingress/security/basic-auth.md)
  - [TLS Authentication](/docs/tasks/ingress/security/tls-auth.md)
  - [Configuring RBAC](/docs/tasks/ingress/security/rbac.md)
  - [Running Voyager per Namespace](/docs/tasks/ingress/security/restrict-namespace.md)
- Monitoring
  - [Exposing HAProxy Stats](/docs/tasks/ingress/monitoring/stats-and-prometheus.md)
- [Replicas and Horizontal Pod Autoscaling](/docs/tasks/ingress/replicas-and-autoscaling.md)
- [Placement of HAProxy Pods](/docs/tasks/ingress/pod-placement.md)
- [Debugging Ingress](/docs/tasks/ingress/debugging.md)


## Certificate
Voyager can automaticallty provision and refresh SSL certificates issued from Let's Encrypt using a custom Kubernetes [Certificate](/docs/tasks/certificate) resource.

**Features**
- Provision free TLS certificates from Let's Encrypt,
- Manage issued certificates using a Kubernetes Third Party Resource,
- Domain validation using ACME dns-01 challenges,
- Support for multiple DNS providers,
- Auto Renew Certificates,
- Use issued Certificates with Ingress to Secure Communications.
