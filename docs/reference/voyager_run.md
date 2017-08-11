## voyager run

Run operator

### Synopsis


Run operator

```
voyager run [flags]
```

### Options

```
      --address string                        Address to listen on for web interface and telemetry. (default ":56790")
      --analytics                             Send analytical event to Google Analytics (default true)
      --cloud-config string                   The path to the cloud provider configuration file.  Empty string for no configuration file.
  -c, --cloud-provider string                 Name of cloud provider
      --haproxy-image string                  haproxy image name to be run (default "appscode/haproxy:1.7.6-3.1.0")
      --haproxy.server-metric-fields string   Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1 (default "2,3,4,5,6,7,8,9,13,14,15,16,17,18,21,24,33,35,38,39,40,41,42,43,44")
      --haproxy.timeout duration              Timeout for trying to get stats from HAProxy. (default 5s)
  -h, --help                                  help for run
      --http-challenge-port int               Port used to answer ACME HTTP challenge (default 56791)
      --ingress-class string                  Ingress class handled by voyager. Unset by default. Set to voyager to only handle ingress with annotation kubernetes.io/ingress.class=voyager.
      --kubeconfig string                     Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string                         The address of the Kubernetes API server (overrides any value in kubeconfig)
      --operator-service string               Name of service used to expose voyager operator (default "voyager-operator")
      --rbac                                  Enable RBAC for operator & offshoot Kubernetes objects
      --resync-period duration                If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out. (default 2m0s)
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [voyager](voyager.md)	 - Voyager by Appscode - Secure Ingress Controller for Kubernetes

