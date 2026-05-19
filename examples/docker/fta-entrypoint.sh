#!/bin/bash

THREADS=$(grep -c ^processor /proc/cpuinfo)
sed -i "s/pfls: 15/pfls: $THREADS/" /etc/pftool/etc/pftool.cfg
sed -i "s/pfcp: 15/pfcp: $THREADS/" /etc/pftool/etc/pftool.cfg
sed -i "s/pfcm: 15/pfcm: $THREADS/" /etc/pftool/etc/pftool.cfg


rm /run/nologin
/usr/sbin/sshd -t
/usr/sbin/sshd

/usr/sbin/conduit-runner -d
