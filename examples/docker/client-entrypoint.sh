#!/bin/bash
set -euo pipefail

cp /etc/conduit/keys/conduit-external-ca.pem /etc/ssl/certs/conduit-external-ca.pem
update-ca-certificates --verbose --fresh

echo 'password' | sudo -u testuser kinit testuser

if [ "$#" -gt 0 ]; then
    exec sudo -u testuser -H -- "$@"
fi

exec sudo -u testuser -H /bin/bash
