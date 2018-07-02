FROM haproxy:1.8.12

ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN true

# Installs required packages
# Change timezone to UTC
RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates runit lua5.3 lua-socket \
  && rm -rf /var/lib/apt/lists/* /usr/share/doc /usr/share/man /tmp/* /etc/sv /etc/service \
  && echo 'Etc/UTC' > /etc/timezone

# Install socklog
COPY socklog.deb .
RUN set -x && apt install ./socklog.deb && rm socklog.deb

ENV TZ     :/etc/localtime
ENV LANG   en_US.utf8

COPY voyager /usr/bin/voyager
COPY auth-request.lua /etc/auth-request.lua

# Setup runit scripts
COPY sv /etc/sv/
RUN ln -s /etc/sv /etc/service

COPY runit.sh /runit.sh
ENTRYPOINT ["/runit.sh"]
