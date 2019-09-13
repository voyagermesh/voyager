---
title: Voyager Check
menu:
  product_voyager_{{ .version }}:
    identifier: voyager-check
    name: Voyager Check
    parent: reference
product_name: voyager
menu_name: product_voyager_{{ .version }}
section_menu_id: reference
---
## voyager check

Check Ingress

### Synopsis

Check Ingress

```
voyager check [flags]
```

### Options

```
  -c, --cloud-provider string   Name of cloud provider
      --from-file string        YAML formatted file containing ingress
  -h, --help                    help for check
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --enable-analytics                 Send analytical events to Google Analytics (default true)
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

* [voyager](/docs/reference/voyager.md)	 - Voyager by Appscode - Secure HAProxy Ingress Controller for Kubernetes

