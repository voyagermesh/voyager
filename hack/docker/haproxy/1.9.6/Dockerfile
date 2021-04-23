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

FROM haproxy:1.9.6

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
