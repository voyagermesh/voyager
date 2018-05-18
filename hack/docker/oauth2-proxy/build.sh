#!/bin/bash
set -x
set -euo pipefail

wget https://github.com/bitly/oauth2_proxy/releases/download/v2.2/oauth2_proxy-2.2.0.linux-amd64.go1.8.1.tar.gz \
  && tar -xzvf oauth2_proxy-2.2.0.linux-amd64.go1.8.1.tar.gz \
  && chmod +x oauth2_proxy-2.2.0.linux-amd64.go1.8.1/oauth2_proxy

docker build -t appscode/oauth2_proxy:2.2.0 .
docker push appscode/oauth2_proxy:2.2.0

rm -rf oauth2_proxy-2.2.0.linux-amd64.go1.8.1 oauth2_proxy-2.2.0.linux-amd64.go1.8.1.tar.gz
