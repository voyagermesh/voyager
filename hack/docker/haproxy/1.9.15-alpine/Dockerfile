# Copyright AppsCode Inc. and Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM debian:buster

ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN true

RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends apt-transport-https ca-certificates curl

RUN set -x \
  && curl -fsSL -o auth-request.lua https://raw.githubusercontent.com/appscode/haproxy-auth-request/v1.9.15/auth-request.lua



FROM haproxy:1.9.15-alpine

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

COPY  bin/{ARG_OS}_{ARG_ARCH}/{ARG_BIN} /usr/bin/{ARG_BIN}
COPY --from=0 /auth-request.lua /etc/auth-request.lua

# Setup runit scripts
COPY hack/docker/haproxy/1.9.15-alpine/sv /etc/sv/
RUN ln -s /etc/sv /etc/service

COPY hack/docker/haproxy/1.9.15-alpine/runit.sh /runit.sh
ENTRYPOINT ["/runit.sh"]
