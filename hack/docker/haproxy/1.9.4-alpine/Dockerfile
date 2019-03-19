FROM haproxy:1.9.4-alpine

# Installs required packages
# Change timezone to UTC
RUN set -x \
  && apk add --update --no-cache ca-certificates su-exec runit socklog tzdata bash openrc lua5.3 lua-socket \
  && rm -rf /etc/sv /etc/service \
  && echo 'Etc/UTC' > /etc/timezone \
  && ln -sf /usr/share/lua/ /usr/local/share/ \
  && ln -sf /usr/lib/lua/ /usr/local/lib/

ENV TZ     :/etc/localtime
ENV LANG   en_US.utf8

COPY voyager /usr/bin/voyager
COPY auth-request.lua /etc/auth-request.lua

# Setup runit scripts
COPY sv /etc/sv/
RUN ln -s /etc/sv /etc/service

COPY runit.sh /runit.sh
ENTRYPOINT ["/runit.sh"]
