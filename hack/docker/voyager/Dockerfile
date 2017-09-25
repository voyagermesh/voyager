FROM alpine

RUN set -x \
  && apk add --update --no-cache ca-certificates

COPY templates /srv/voyager/templates/
COPY voyager /usr/bin/voyager

ENTRYPOINT ["voyager"]
