#!/bin/bash

THREADS=$(nproc)
PFCP_THREADS=$((THREADS > 2 ? THREADS - 2 : 1))

sed -i -E \
    -e "s/^(pfls:[[:space:]]*)[0-9]+/\1${PFCP_THREADS}/" \
    -e "s/^(pfcp:[[:space:]]*)[0-9]+/\1${PFCP_THREADS}/" \
    -e "s/^(pfcm:[[:space:]]*)[0-9]+/\1${PFCP_THREADS}/" \
    /etc/pftool/etc/pftool.cfg

rm -f /run/nologin
/usr/sbin/sshd -t
/usr/sbin/sshd

/usr/sbin/conduit-runner -d
