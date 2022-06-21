---
title: OAuth2 Authentication | Kubernetes Ingress
menu:
  docs_{{ .version }}:
    identifier: oauth2-security
    name: OAuth2
    parent: security-ingress
    weight: 20
product_name: voyager
menu_name: docs_{{ .version }}
section_menu_id: guides
---

> New to Voyager? Please start [here](/docs/concepts/overview.md).

# OAuth2 Authentication

You can configure [external authentication / oauth](https://oauth.net/2/) on Voyager Ingress controller via `frontendrules`. For this you have to configure and expose [oauth2-proxy](https://github.com/bitly/oauth2_proxy) and specify it as a backend under same host. For example:

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: default
spec:
  frontendRules:
  - port: 80
    auth:
      oauth:
      - host: voyager.appscode.ninja
        authBackend: auth-be
        authPath: /oauth2/auth
        signinPath: /oauth2/start
        paths: 
        - /app
  rules:
  - host: voyager.appscode.ninja
    http:
      paths:
      - path: /health
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /app
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /oauth2
        backend:
          name: auth-be
          service:
            name: oauth2-proxy
            port:
              number: 4180
```

Please note the followings:

- Oauth will be enabled only for the specified paths. It is not necessary that this paths should match with the paths specified in the http-rules.

- Auth backend and app backend should be under same host.

- For secure/tls connections, you have to set `cookie-secure=true` (default) and for insecure/non-tls connections, you have to set `cookie-secure=false` while configuring `oauth2-proxy`.

- You can use any random string as `OAUTH2_PROXY_COOKIE_SECRET` while configuring `oauth2-proxy`. You can generate one using following command:

```bash
$ python -c 'import os,base64; print base64.b64encode(os.urandom(16))'
```
 
- If you use standard ports, you have to write frontend rules under port `80` for non-tls and under port `443` for tls.

- You can not use different auth backends for different paths under same host and port. However, it is possible to configure different auth backends for different hosts under same port. For example:

```yaml
apiVersion: voyager.appscode.com/v1
kind: Ingress
metadata:
  name: auth-ingress
  namespace: default
spec:
  frontendRules:
  - port: 80
    auth:
      oauth:
      - host: team01.example.com
        authBackend: google-auth
        authPath: /google/auth
        signinPath: /google/start
        paths:
        - /foo
      - host: team02.example.com
        authBackend: github-auth
        authPath: /github/auth
        signinPath: /github/start
        paths:
        - /bar
  rules:
  - host: team01.example.com
    http:
      paths:
      - path: /foo
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /google
        backend:
          name: google-auth
          service:
            name: oauth2-proxy-google
            port:
              number: 4180
  - host: team02.example.com
    http:
      paths:
      - path: /bar
        backend:
          service:
            name: test-server
            port:
              number: 80
      - path: /github
        backend:
          name: github-auth
          service:
            name: oauth2-proxy-github
            port:
              number: 4180
```

## Next Steps

- Learn how to configure GitHub as auth provider [here](oauth-github.md).
- Learn how to configure Google as auth provider [here](oauth-google.md).
- Learn how to secure Kubernetes Dashboard using voyager external auth [here](oauth-dashboard.md).
