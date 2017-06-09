package template

const HAProxyTemplate = `# HAProxy configuration generated by https://github.com/appscode/voyager
# DO NOT EDIT!

global
    daemon
    stats socket /tmp/haproxy
    server-state-file global
    server-state-base /var/state/haproxy/
    maxconn 4000
    # log using a syslog socket
    log /dev/log local0 info
    log /dev/log local0 notice
    {% if SSLCert %}
    tune.ssl.default-dh-param 2048
    ssl-default-bind-ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-DSS-AES128-GCM-SHA256:kEDH+AESGCM:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA:ECDHE-ECDSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES128-SHA:DHE-DSS-AES128-SHA256:DHE-RSA-AES256-SHA256:DHE-DSS-AES256-SHA:DHE-RSA-AES256-SHA:!aNULL:!eNULL:!EXPORT:!DES:!RC4:!3DES:!MD5:!PSK
    {% endif %}

defaults
    log global

    option http-server-close

    # Disable logging of null connections (haproxy connections like checks).
    # This avoids excessive logs from haproxy internals.
    option dontlognull

    # Maximum time to wait for a connection attempt to a server to succeed.
    timeout connect         50000

    # Maximum inactivity time on the client side.
    # Applies when the client is expected to acknowledge or send data.
    timeout client          50000

    # Inactivity timeout on the client side for half-closed connections.
    # Applies when the client is expected to acknowledge or send data
    # while one direction is already shut down.
    timeout client-fin      50000

    # Maximum inactivity time on the server side.
    timeout server          50000

    # timeout to use with WebSocket and CONNECT
    timeout tunnel          50000

    # default traffic mode is http
    # mode is overwritten in case of tcp services
    mode http

    # errorloc 400 https://appscode.com/errors/400
    # errorloc 403 https://appscode.com/errors/403
    # errorloc 408 https://appscode.com/errors/408
    # errorloc 500 https://appscode.com/errors/500
    # errorloc 502 https://appscode.com/errors/502
    # errorloc 503 https://appscode.com/errors/503
    # errorloc 504 https://appscode.com/errors/504

{% for name, resolver in DNSResolvers %}
resolvers {{ name }}
    {% for ns in resolver.nameserver %}
    nameserver dns{{ loop.index }} {{ ns }}
    {% endfor %}
    {% if resolver.retries|integer %}
    resolve_retries {{ resolver.retries|integer }}
    {% endif %}
    {% for event, time in resolver.timeout %}
    timeout {{ event }} {{ time }}
    {% endfor %}
    {% for status, period in resolver.hold %}
    hold {{ status }} {{ period }}
    {% endfor %}
{% endfor %}

{% if Stats %}
listen stats
    bind *:{{ StatsPort }}
    mode http
    stats enable
    stats realm Haproxy\ Statistics
    stats uri /
    {% if StatsUserName %}stats auth {{ StatsUserName }}:{{ StatsPassWord }}{% endif %}
{% endif %}

{% if DefaultBackend %}
# default backend
backend default-backend
    {% if Sticky %}cookie SERVERID insert indirect nocache{% endif %}

    {% for rule in DefaultBackend.BackendRules %}
    {{ rule }}
    {% endfor %}

    {% for rule in DefaultBackend.RewriteRules %}
    reqrep {{ rule }}
    {% endfor %}

    {% for rule in DefaultBackend.HeaderRules %}
    acl ___header_x_{{ forloop.Counter }}_exists req.hdr({{ rule|header_name }}) -m found
    http-request add-header {{ rule }} unless ___header_x_{{ forloop.Counter }}_exists
    {% endfor %}

    {% for e in DefaultBackend.Endpoints %}
    {% if e.ExternalName %}
    {% if e.UseDNSResolver %}
    server {{ e.Name }} {{ e.ExternalName }}:{{ e.Port }} resolve-prefer ipv4 {% if e.DNSResolver %} check resolvers {{ e.DNSResolver }} {% endif %}
    {% elif not svc.Backends.BackendRules %}
    acl https ssl_fc
    http-request redirect location https://{{e.ExternalName}}:{{ e.Port }} code 301 if https
    http-request redirect location http://{{e.ExternalName}}:{{ e.Port }} code 301 unless https
    {% endif %}
    {% else %}
    server {{ e.Name }} {{ e.IP }}:{{ e.Port }} {% if e.Weight %}weight {{ e.Weight|integer }} {% endif %} {% if Sticky %}cookie {{ e.Name }} {% endif %}
    {% endif %}
    {% endfor %}
{% endif %}

{% if HttpsService %}
# https service
frontend https-frontend
    bind *:443 {% if AcceptProxy %}accept-proxy{% endif %} ssl no-sslv3 no-tlsv10 no-tls-tickets crt /etc/ssl/private/haproxy/ alpn http/1.1
    # Mark all cookies as secure
    rsprep ^Set-Cookie:\ (.*) Set-Cookie:\ \1;\ Secure
    # Add the HSTS header with a 6 month max-age
    rspadd  Strict-Transport-Security:\ max-age=15768000

    mode http
    option httplog
    option forwardfor

{% for svc in HttpsService %}
    {% set both = 0 %}
    {% if svc.AclMatch %}acl url_acl_{{ svc.Name }} path_beg {{ svc.AclMatch }} {% set both = both + 1 %}{% endif %}
    {% if svc.Host %}acl host_acl_{{ svc.Name }} {{ svc.Host|host_name }} {% set both = both + 1 %}{% endif %}
    use_backend https-{{ svc.Name }} {% if both != 0 %}if {% endif %}{% if svc.AclMatch %}url_acl_{{ svc.Name }}{% endif %} {% if svc.Host %}host_acl_{{ svc.Name }}{% endif %}
{% endfor %}
    {% if DefaultBackend %}default_backend default-backend{% endif %}
{% endif %}

{% for svc in HttpsService %}
backend https-{{ svc.Name }}
    {% if Sticky %}cookie SERVERID insert indirect nocache{% endif %}

    {% for rule in svc.Backends.BackendRules %}
    {{ rule }}
    {% endfor %}

    {% for rule in svc.Backends.RewriteRules %}
    reqrep {{ rule }}
    {% endfor %}

    {% for rule in svc.Backends.HeaderRules %}
    acl ___header_x_{{ forloop.Counter }}_exists req.hdr({{ rule|header_name }}) -m found
    http-request add-header {{ rule }} unless ___header_x_{{ forloop.Counter }}_exists
    {% endfor %}

    {% for e in svc.Backends.Endpoints %}
    {% if e.ExternalName %}
    {% if e.UseDNSResolver %}
    server {{ e.Name }} {{ e.ExternalName }}:{{ e.Port }} resolve-prefer ipv4 {% if e.DNSResolver %} check resolvers {{ e.DNSResolver }} {% endif %}
    {% elif not svc.Backends.BackendRules %}
    http-request redirect location https://{{e.ExternalName}}:{{ e.Port }} code 301
    {% endif %}
    {% else %}
    server {{ e.Name }} {{ e.IP }}:{{ e.Port }} {% if e.Weight %}weight {{ e.Weight|integer }} {% endif %} {% if Sticky %} cookie {{ e.Name }} {% endif %}
    {% endif %}
    {% endfor %}
{% endfor %}

{% if HttpService %}
# http services.
frontend http-frontend
    bind *:80 {% if AcceptProxy %}accept-proxy{% endif %}
    mode http
    option httplog
    option forwardfor

{% for svc in HttpService %}
    {% set both = 0 %}
    {% if svc.AclMatch %}acl url_acl_{{ svc.Name }} path_beg {{ svc.AclMatch }} {% set both = both + 1 %}{% endif %}
    {% if svc.Host %}acl host_acl_{{ svc.Name }} {{ svc.Host|host_name }} {% set both = both + 1 %}{% endif %}
    use_backend http-{{ svc.Name }} {% if both != 0 %}if {% endif %}{% if svc.AclMatch %}url_acl_{{ svc.Name }}{% endif %} {% if svc.Host %}host_acl_{{ svc.Name }}{% endif %}
{% endfor %}
    {% if DefaultBackend %}default_backend default-backend{% endif %}
{% endif %}

{% for svc in HttpService %}
backend http-{{ svc.Name }}
    {% if Sticky %}cookie SERVERID insert indirect nocache{% endif %}

    {% for rule in svc.Backends.BackendRules %}
    {{ rule }}
    {% endfor %}

    {% for rule in svc.Backends.RewriteRules %}
    reqrep {{ rule }}
    {% endfor %}

    {% for rule in svc.Backends.HeaderRules %}
    acl ___header_x_{{ forloop.Counter }}_exists req.hdr({{ rule|header_name }}) -m found
    http-request add-header {{ rule }} unless ___header_x_{{ forloop.Counter }}_exists
    {% endfor %}

    {% for e in svc.Backends.Endpoints %}
    {% if e.ExternalName %}
    {% if e.UseDNSResolver %}
    server {{ e.Name }} {{ e.ExternalName }}:{{ e.Port }} resolve-prefer ipv4 {% if e.DNSResolver %} check resolvers {{ e.DNSResolver }} {% endif %}
    {% elif not svc.Backends.BackendRules %}
    http-request redirect location http://{{e.ExternalName}}:{{ e.Port }} code 301
    {% endif %}
    {% else %}
    server {{ e.Name }} {{ e.IP }}:{{ e.Port }} {% if e.Weight %}weight {{ e.Weight|integer }} {% endif %} {% if Sticky %}cookie {{ e.Name }} {% endif %}
    {% endif %}
    {% endfor %}
{% endfor %}


{% if TCPService %}
# tcp service
{% for svc in TCPService %}
frontend tcp-frontend-key-{{ svc.Port }}
    bind *:{{ svc.Port }} {% if AcceptProxy %}accept-proxy{% endif %} {% if svc.SecretName %}ssl no-sslv3 no-tlsv10 no-tls-tickets crt /etc/ssl/private/haproxy/{{ svc.SecretName }}.pem{% endif %} {%if svc.ALPNOptions %} {{svc.ALPNOptions}}{% endif %}
    mode tcp
    default_backend tcp-{{ svc.Name }}
{% endfor %}
{% endif %}

{% for svc in TCPService %}
backend tcp-{{ svc.Name }}
    mode tcp

    {% for rule in svc.Backends.BackendRules %}
    {{ rule }}
    {% endfor %}

    {% if Sticky %}
    stick-table type ip size 100k expire 30m
    stick on src
    {% endif %}

    {% for e in svc.Backends.Endpoints %}
    {% if e.ExternalName and e.UseDNSResolver %}
    server {{ e.Name }} {{ e.ExternalName }}:{{ e.Port }} resolve-prefer ipv4 {% if e.DNSResolver %} check resolvers {{ e.DNSResolver }} {% endif %}
    {% else %}
    server {{ e.Name }} {{ e.IP }}:{{ e.Port }} {% if e.Weight %}weight {{ e.Weight|integer }} {% endif %}
    {% endif %}
    {% endfor %}
{% endfor %}

{% if !HttpService and !HttpsService and DefaultBackend %}
frontend http-frontend
    bind *:80 {% if AcceptProxy %}accept-proxy{% endif %}
    mode http

    option forwardfor
    default_backend default-backend
{% endif %}`
