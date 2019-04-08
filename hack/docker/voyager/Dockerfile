FROM haproxy:1.9.6-alpine

RUN set -x \
  && apk add --update --no-cache ca-certificates lua5.3 lua-socket \
  && ln -sf /usr/share/lua/ /usr/local/share/ \
  && ln -sf /usr/lib/lua/ /usr/local/lib/

COPY auth-request.lua /etc/auth-request.lua
COPY templates /srv/voyager/templates/
COPY voyager /usr/bin/voyager

# https://github.com/appscode/voyager/pull/1038
COPY test.pem /etc/ssl/private/haproxy/tls/test.pem
COPY errorfiles /srv/voyager/errorfiles/

ENTRYPOINT ["voyager"]
