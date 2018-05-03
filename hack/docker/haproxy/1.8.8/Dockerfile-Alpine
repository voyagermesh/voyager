FROM haproxy:1.8.8-alpine

# Installs runit and socklog
# make the "en_US.UTF-8" locale so postgres will be utf-8 enabled by default
# change timezone to UTC

RUN set -x \
  && apk add --update --no-cache ca-certificates su-exec runit socklog tzdata bash openrc lua5.3 lua-socket \
  && rm -rf /etc/sv /etc/service \
  && echo 'Etc/UTC' > /etc/timezone \
  && ln -sf /usr/share/lua/ /usr/local/share/ \
  && ln -sf /usr/lib/lua/ /usr/local/lib/

ENV TZ     :/etc/localtime
ENV LANG   en_US.utf8

COPY voyager /usr/bin/voyager

# Setup runit scripts
COPY sv /etc/sv/
RUN ln -s /etc/sv /etc/service

COPY auth-request.lua /etc/auth-request.lua

COPY runit.sh /runit.sh
ENTRYPOINT ["/runit.sh"]
