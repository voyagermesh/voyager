FROM alpine

RUN set -x \
  && apk add --update --no-cache ca-certificates

COPY oauth2_proxy-2.2.0.linux-amd64.go1.8.1/oauth2_proxy /bin/oauth2_proxy

ENTRYPOINT ["oauth2_proxy"]
