#!/bin/bash

export HAPROXY_CONTROLLER_ARGS="$@"
export > /etc/envvars

[[ $DEBUG == true ]] && set -x

# create haproxy.cfg dir
mkdir /etc/haproxy
touch /var/run/haproxy.pid

echo "Syncing HAProxy controller ..."
mkdir -p /etc/ssl/private/haproxy
cmd="voyager haproxy-controller --init-only $HAPROXY_CONTROLLER_ARGS"
echo $cmd
$cmd
rc=$?; if [[ $rc != 0 ]]; then exit $rc; fi

echo "Starting runit..."
exec /sbin/runsvdir -P /etc/service
