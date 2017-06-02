#!/bin/bash

export KLOADER_ARGS="$@"
export > /etc/envvars

[[ $DEBUG == true ]] && set -x

# create haproxy.cfg dir
mkdir /etc/haproxy

CERT_DIR=/etc/ssl/private/haproxy
mkdir -p /etc/ssl/private/haproxy

# http://stackoverflow.com/a/2108296
for dir in /srv/haproxy/secrets/*/
do
	# remove trailing /
	dir=${dir%*/}
	# just basename
	secret=${dir##*/}

	cat $dir/tls.crt >  $CERT_DIR/$secret.pem
	cat $dir/tls.key >> $CERT_DIR/$secret.pem
done

echo "Checking HAProxy configuration ..."
cmd="exec /kloader check $KLOADER_ARGS"
echo $cmd
$cmd
rc=$?; if [[ $rc != 0 ]]; then exit $rc; fi

echo "Starting runit..."
exec /usr/sbin/runsvdir-start
