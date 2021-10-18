---
title: Voyager Coordinator
menu:
  docs_{{ .version }}:
    identifier: voyager-coordinator
    name: Voyager Coordinator
    parent: reference-operator
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: reference
---
## voyager coordinator

Synchronizes HAProxy config

```
voyager coordinator [command] [flags]
```

### Options

```
      --burst int                    The maximum burst for throttle (default 1000000)
      --cert-dir string              Path where tls certificates are stored for HAProxy (default "/etc/ssl/private/haproxy")
  -c, --cloud-provider string        Name of cloud provider
      --config-dir string            Path where HAProxy config is stored (default "/shared/etc/haproxy")
  -h, --help                         help for coordinator
      --ingress-api-version string   API version of ingress resource
      --ingress-name string          Name of ingress resource
      --kubeconfig string            Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string                The address of the Kubernetes API server (overrides any value in kubeconfig)
      --qps float32                  The maximum QPS to the master from this client (default 1e+06)
      --resync-period duration       If non-zero, will re-list this often. Otherwise, re-list will be delayed as long as possible (until the upstream source closes the watch or times out. (default 10m0s)
```

### Options inherited from parent commands

```
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
```

### SEE ALSO

* [voyager](/docs/reference/operator/voyager.md)	 - Voyager by AppsCode - Secure L7/L4 Ingress Controller for Kubernetes

