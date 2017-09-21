## voyager

Voyager by Appscode - Secure Ingress Controller for Kubernetes

### Synopsis


Voyager by Appscode - Secure Ingress Controller for Kubernetes

### Options

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Analytics (default true)
  -h, --help                             help for voyager
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
* [voyager check](voyager_check.md)	 - Check Ingress
* [voyager export](voyager_export.md)	 - Export Prometheus metrics for HAProxy
* [voyager kloader](voyager_kloader.md)	 - Reloads HAProxy when configmap changes
* [voyager run](voyager_run.md)	 - Run operator
* [voyager tls-mounter](voyager_tls-mounter.md)	 - Mounts TLS certificates in HAProxy pods
* [voyager version](voyager_version.md)	 - Prints binary version number.

