---
title: Voyager
menu:
  docs_{{ .version }}:
    identifier: voyager
    name: Voyager
    parent: reference-operator
    weight: 0

product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: reference
url: /docs/{{ .version }}/reference/operator/
aliases:
- /docs/{{ .version }}/reference/operator/voyager/
---
## voyager

Voyager by AppsCode - Secure L7/L4 Ingress Controller for Kubernetes

### Options

```
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
  -h, --help                             help for voyager
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
```

### SEE ALSO

* [voyager coordinator](/docs/reference/operator/voyager_coordinator.md)	 - Synchronizes HAProxy config
* [voyager init](/docs/reference/operator/voyager_init.md)	 - Initialize HAProxy config
* [voyager operator](/docs/reference/operator/voyager_operator.md)	 - Launch Voyager Ingress Operator
* [voyager run](/docs/reference/operator/voyager_run.md)	 - Launch Voyager Ingress Webhook Server
* [voyager version](/docs/reference/operator/voyager_version.md)	 - Prints binary version number.

