#!/bin/bash -e
config=$(mktemp -t)
if test -z "$config" ; then
	echo error creating temporary file for configuration
	exit 1
fi
trap 'rm -f "$config"' EXIT
cat > "$config" << EOF
[req]
prompt=no
distinguished_name=dn
x509_extensions=extensions
[extensions]
keyUsage=critical,digitalSignature,keyEncipherment,keyCertSign
extendedKeyUsage=serverAuth,clientAuth
basicConstraints=critical,CA:TRUE
subjectAltName=DNS:localhost,email:a@a.com
[dn]
O=Acme Co
EOF
serial=$(dd if=/dev/random bs=1 count=16 status=none | hexdump -e '"%x1"')
openssl req -new -set_serial 0x"$serial" -x509 -sha512 -days 365 -key cert.key -config "$config" -out cert.cert
cp cert.cert cacert.crt
