#!/bin/bash

export > /etc/envvars

[[ $DEBUG == true ]] && set -x

# propagate kloader args
sed -i "s|__KLOADER_ARGS__|$@|g" /etc/sv/reloader/run

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

echo "Starting runit..."
exec /usr/sbin/runsvdir-start
