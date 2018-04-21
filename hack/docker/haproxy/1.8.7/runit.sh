#!/bin/bash

export HAPROXY_CONTROLLER_ARGS="$@"
export > /etc/envvars

[[ $DEBUG == true ]] && set -x

# create haproxy.cfg dir
mkdir /etc/haproxy
touch /var/run/haproxy.pid
mkdir -p /etc/ssl/private/haproxy

echo "Starting runit..."
exec /sbin/runsvdir -P /etc/service
