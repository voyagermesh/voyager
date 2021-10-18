---
title: Voyager Operator
menu:
  docs_{{ .version }}:
    identifier: voyager-operator
    name: Voyager Operator
    parent: reference-operator
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: reference
---
## voyager operator

Launch Voyager Ingress Operator

### Synopsis

Launch Voyager Ingress Operator

```
voyager operator [flags]
```

### Options

```
      --burst int                  The maximum burst for throttle (default 1000000)
      --cloud-config string        The path to the cloud provider configuration file.  Empty string for no configuration file.
      --cloud-provider string      Name of cloud provider
      --coordinator-image string   HAProxy sidecar Docker image
      --custom-templates string    Glob pattern of custom HAProxy template files used to override built-in templates
      --haproxy-image string       HAProxy Docker image (default "appscode/haproxy:2.4.4-alpine")
  -h, --help                       help for operator
      --ingress-class string       Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.
      --kubeconfig string          Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --license-file string        Path to license file
      --master string              The address of the Kubernetes API server (overrides any value in kubeconfig)
      --qps float                  The maximum QPS to the master from this client (default 1e+06)
      --resync-period duration     If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out. (default 10m0s)
      --validate-haproxy-config    If true, validates generated haproxy.cfg before sending to HAProxy pods. (default true)
```

### Options inherited from parent commands

```
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
```

### SEE ALSO

* [voyager](/docs/reference/operator/voyager.md)	 - Voyager by AppsCode - Secure L7/L4 Ingress Controller for Kubernetes

