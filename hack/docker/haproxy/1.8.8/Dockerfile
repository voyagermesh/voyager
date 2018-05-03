FROM haproxy:1.8.8

ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN true

# Installs runit and socklog
# make the "en_US.UTF-8" locale so postgres will be utf-8 enabled by default
# change timezone to UTC
# `socklog` not avaiable for `stretch`, use `jessie` instead

RUN set -x \
  && echo "deb http://deb.debian.org/debian jessie main" >> /etc/apt/sources.list \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata runit socklog lua5.3 lua-socket \
  && rm -rf /var/lib/apt/lists/* /usr/share/doc /usr/share/man /tmp/* /etc/sv /etc/service \
  && echo 'Etc/UTC' > /etc/timezone

ENV TZ     :/etc/localtime
ENV LANG   en_US.utf8

COPY voyager /usr/bin/voyager

# Setup runit scripts
COPY sv /etc/sv/
RUN ln -s /etc/sv /etc/service

COPY auth-request.lua /etc/auth-request.lua

COPY runit.sh /runit.sh
ENTRYPOINT ["/runit.sh"]
