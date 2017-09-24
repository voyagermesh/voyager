#!/bin/bash

export TLS_MOUNTER_ARGS="$1 $2 $3 $4 $5"
# http://wiki.bash-hackers.org/scripting/posparams#shifting
shift 3
export KLOADER_ARGS="$@"
export > /etc/envvars

[[ $DEBUG == true ]] && set -x

# create haproxy.cfg dir
mkdir /etc/haproxy

echo "Mounting TLS certificates ..."
mkdir -p /etc/ssl/private/haproxy
cmd="voyager tls-mounter $TLS_MOUNTER_ARGS"
echo $cmd
$cmd
rc=$?; if [[ $rc != 0 ]]; then exit $rc; fi

echo "Checking HAProxy configuration ..."
cmd="voyager kloader check $KLOADER_ARGS"
echo $cmd
$cmd
rc=$?; if [[ $rc != 0 ]]; then exit $rc; fi

echo "Starting runit..."
exec /usr/sbin/runsvdir-start
