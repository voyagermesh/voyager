---
title: Voyager Version
menu:
  docs_{{ .version }}:
    identifier: voyager-version
    name: Voyager Version
    parent: reference-operator
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: reference
---
## voyager version

Prints binary version number.

```
voyager version [flags]
```

### Options

```
      --check string   Check version constraint
  -h, --help           help for version
      --short          Print just the version number.
```

### Options inherited from parent commands

```
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
```

### SEE ALSO

* [voyager](/docs/reference/operator/voyager.md)	 - Voyager by AppsCode - Secure L7/L4 Ingress Controller for Kubernetes

