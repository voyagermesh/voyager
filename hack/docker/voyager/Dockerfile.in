FROM {ARG_FROM}

RUN set -x \
  && apk add --update --no-cache ca-certificates lua5.3 lua-socket \
  && ln -sf /usr/share/lua/ /usr/local/share/ \
  && ln -sf /usr/lib/lua/ /usr/local/lib/

COPY bin/auth-request.lua /etc/auth-request.lua
COPY hack/docker/voyager/templates /srv/voyager/templates/
ADD bin/{ARG_OS}_{ARG_ARCH}/{ARG_BIN} /{ARG_BIN}

# https://github.com/appscode/voyager/pull/1038
COPY hack/docker/voyager/test.pem /etc/ssl/private/haproxy/tls/test.pem
COPY hack/docker/voyager/errorfiles /srv/voyager/errorfiles/

ENTRYPOINT ["/{ARG_BIN}"]
