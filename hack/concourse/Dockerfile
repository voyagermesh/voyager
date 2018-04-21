FROM golang:1.10-alpine

RUN apk update && \
    apk add git python py-pip && \
    pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage && \
    pip install pyyaml
