## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building voyager.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of voyager should not change.

### Building voyager on local environment
Some of the voyager development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

### Go development environment
voyager is written in the go programming language. The release is built and tested on **go 1.7.5**. If you haven't set up a Go
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install the go tools.

### Dependency management
voyager build and test scripts use glide to manage dependencies.

To install glide follow [these instructions](https://github.com/Masterminds/glide#install).

Currently the project includes all its required dependencies inside `vendor` to make things easier.

### Local Build
To build voyager using your local Go development environment (generate linux binaries):
```sh
$ ./hack/make.py build voyager
```
Read full [Build instructions](build.md).

<br><br>
## Architecture
voyager works by implementing third party resource data watcher for kubernetes. It connects with k8s apiserver
for specific events as ADD, UPDATE and DELETE. and perform required operations.

Ingress watcher generates the configuration for HAProxy and stores it as a ConfigMaps and creates a RC with
specified HAProxy - that is configured with auto reload while any changes happens to ConfigMap data. This is handled via
[kloader](https://github.com/appscode/kloader). Voyager keeps the ingress resource and the configuration in sync
by performing processing on the resources.

Certificate watcher watch and process certificates third party data and obtain a ACME certificates.
