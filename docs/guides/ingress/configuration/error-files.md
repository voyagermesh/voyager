---
title: Configure Ingress Error Files
menu:
  docs_{{ .version }}:
    identifier: error-files-configuration
    name: Error Files
    parent: config-ingress
    weight: 10
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Error Files

Using voyager you can configure haproxy to return a file-content or, execute a command instead of returning generated errors. To achieve this you need to create a `configmap` specifying the file-content or, command for different status codes. Then you have to specify the `configmap` name using `ingress.appscode.com/errorfiles` annotation. Then contents of the configmap will be mounted in the haproxy pod in path `/srv/voyager/errorfiles`.

Supported commands are: `errorfile, errorloc, errorloc302, errorloc303`.
And supported status codes are: `200, 400, 403, 405, 408, 429, 500, 502, 503, 504`.

For example, lets consider a `configmap` with following key-value pairs:

```ini
503.http   : <content of 503.http>
408        : []byte("errorfile /dev/null")
500        : []byte("errorloc https://example.com/500.hlml")
```

It will generate following block in `defaults` section of haproxy.cfg:

```ini
errorfile 503 /srv/voyager/errorfiles/503.http
errorfile 408 /dev/null
errorloc  500 https://example.com/500.hlml
```

Note that, when status code with `.http` suffix is used as key, the command will be `errorfile` and you just need to specify the file contents as value.

To learn more about these command see [here](https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#4.2-errorfile).
