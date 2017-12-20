---
title: Voyager Kloader
menu:
  product_voyager_5.0.0-rc.8:
    identifier: voyager-kloader
    name: Voyager Kloader
    parent: reference
product_name: voyager
menu_name: product_voyager_5.0.0-rc.8
section_menu_id: reference
---
## voyager kloader

Reloads HAProxy when configmap changes

### Synopsis

Reloads HAProxy when configmap changes

### Options

```
  -h, --help   help for kloader
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
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [voyager](/docs/reference/voyager.md)	 - Voyager by Appscode - Secure Ingress Controller for Kubernetes
* [voyager kloader check](/docs/reference/voyager_kloader_check.md)	 - Validate kloader configuration
* [voyager kloader run](/docs/reference/voyager_kloader_run.md)	 - Run and hold kloader

