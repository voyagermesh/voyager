FROM haproxy:1.8.8-alpine

RUN set -x \
  && apk add --update --no-cache ca-certificates lua5.3 lua-socket \
  && ln -sf /usr/share/lua/ /usr/local/share/ \
  && ln -sf /usr/lib/lua/ /usr/local/lib/

COPY auth-request.lua /etc/auth-request.lua
COPY templates /srv/voyager/templates/
COPY voyager /usr/bin/voyager

ENTRYPOINT ["voyager"]
