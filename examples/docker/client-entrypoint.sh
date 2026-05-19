#!/bin/bash

cp /etc/conduit/keys/conduit_external_ca.pem /etc/ssl/certs/conduit_external_ca.pem
update-ca-certificates --verbose --fresh

echo 'password' | sudo -u testuser kinit testuser

sudo -u testuser /bin/bash
