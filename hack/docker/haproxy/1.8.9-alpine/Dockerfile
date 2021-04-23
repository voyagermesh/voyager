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

FROM haproxy:1.8.9-alpine

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
