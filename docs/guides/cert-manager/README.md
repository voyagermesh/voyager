---
title: cert-manager | Voyager
menu:
  product_voyager_v11.0.0:
    identifier: readme-cert-manager
    name: Readme
    parent: cert-manager-guides
    weight: -1
product_name: voyager
menu_name: product_voyager_v11.0.0
section_menu_id: guides
url: /products/voyager/v11.0.0/guides/cert-manager/
aliases:
  - /products/voyager/v11.0.0/guides/cert-manager/README/
---

# Guides

Guides show you how to use [jetstack/cert-manager](https://github.com/jetstack/cert-manager) with Voyager to issue free TLS/SSL certificates from Let's Encrypt.

## Features

- Provision free TLS certificates (including wildcard certificates) from Let's Encrypt.
- Manage certificates declaratively using a Kubernetes Custom Resource Definition (CRD).
- Domain validation using ACME http-01 and dns-01 challenges.
- Support for many popular DNS providers.
- Auto Renew certificates.
- Use issued certificates with Ingress to secure communications.

## Next Steps

- [Issue Let's Encrypt certificate using HTTP-01 challenge](/docs/guides/cert-manager/http01_challenge/overview.md)
- DNS-01 challenge providers
  - [Issue Let's Encrypt certificate using AWS Route53](/docs/guides/cert-manager/dns01_challenge/aws-route53.md)
  - [Issue Let's Encrypt certificate using Azure DNS](/docs/guides/cert-manager/dns01_challenge/azure-dns.md)
  - [Issue Let's Encrypt certificate using Google Cloud DNS](/docs/guides/cert-manager/dns01_challenge/google-cloud-dns.md)
  - [Multiple Providers](/docs/guides/cert-manager/dns01_challenge/multiple-challenge-solver.md)
