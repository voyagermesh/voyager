---
title: Voyager Haproxy-Controller
menu:
  product_voyager_6.0.0:
    identifier: voyager-haproxy-controller
    name: Voyager Haproxy-Controller
    parent: reference
product_name: voyager
menu_name: product_voyager_6.0.0
section_menu_id: reference
---
## voyager haproxy-controller

Synchronizes HAProxy config

### Synopsis

Synchronizes HAProxy config

```
voyager haproxy-controller [command] [flags]
```

### Options

```
      --burst int                    The maximum burst for throttle (default 1000000)
      --cert-dir string              Path where tls certificates are stored for HAProxy (default "/etc/ssl/private/haproxy")
  -c, --cloud-provider string        Name of cloud provider
  -h, --help                         help for haproxy-controller
      --ingress-api-version string   API version of ingress resource
      --ingress-name string          Name of ingress resource
      --init-only                    If true, exits after initial tls mount
      --kubeconfig string            Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string                The address of the Kubernetes API server (overrides any value in kubeconfig)
      --qps float32                  The maximum QPS to the master from this client (default 1e+06)
      --resync-period duration       If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out. (default 10m0s)
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Analytics (default true)
      --log.format logFormatFlag         Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true" (default "logger:stderr")
      --log.level levelFlag              Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal] (default "info")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [voyager](/docs/reference/voyager.md)	 - Voyager by Appscode - Secure Ingress Controller for Kubernetes

