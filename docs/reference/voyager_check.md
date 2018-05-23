---
title: Voyager Check
menu:
  product_voyager_7.0.0-rc.3:
    identifier: voyager-check
    name: Voyager Check
    parent: reference
product_name: voyager
menu_name: product_voyager_7.0.0-rc.3
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
      --enable-analytics                 Send analytical events to Google Analytics (default true)
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

* [voyager](/docs/reference/voyager.md)	 - Voyager by Appscode - Secure HAProxy Ingress Controller for Kubernetes

