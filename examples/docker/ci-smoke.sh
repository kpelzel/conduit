#!/bin/sh
set -eu

docker compose up -d --force-recreate --wait --wait-timeout 60 \
  kdc \
  etcd-1 etcd-2 etcd-3 \
  rqlite-1 rqlite-2 rqlite-3 \
  fta1 fta2 \
  conduit-server

docker compose run --rm client sh -lc '
  set -eu
  conduit cp /mnt/fs_1/foo/hello.txt /mnt/fs_2/bar/
  conduit status
  test "$(cat /mnt/fs_2/bar/hello.txt)" = "hello"
'
