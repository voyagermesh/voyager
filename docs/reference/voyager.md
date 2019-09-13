---
title: Voyager
menu:
  product_voyager_{{ .version }}:
    identifier: voyager
    name: Voyager
    parent: reference
    weight: 0

product_name: voyager
menu_name: product_voyager_{{ .version }}
section_menu_id: reference
aliases:
  - /products/voyager/{{ .version }}/reference/

---
## voyager

Voyager by Appscode - Secure HAProxy Ingress Controller for Kubernetes

### Synopsis

Voyager by Appscode - Secure HAProxy Ingress Controller for Kubernetes

### Options

```
      --alsologtostderr                  log to standard error as well as files
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
  -h, --help                             help for voyager
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [voyager check](/docs/reference/voyager_check.md)	 - Check Ingress
* [voyager export](/docs/reference/voyager_export.md)	 - Export Prometheus metrics for HAProxy
* [voyager haproxy-controller](/docs/reference/voyager_haproxy-controller.md)	 - Synchronizes HAProxy config
* [voyager run](/docs/reference/voyager_run.md)	 - Launch Voyager Ingress Controller
* [voyager version](/docs/reference/voyager_version.md)	 - Prints binary version number.

