#!/bin/sh
set -eu

services="
kdc
etcd-1
etcd-2
etcd-3
rqlite-1
rqlite-2
rqlite-3
fta1
fta2
conduit-server
ldap
"

docker compose up -d $services

deadline=$(( $(date +%s) + 180 ))

while :; do
    all_healthy=true

    for service in $services; do
        cid=$(docker compose ps -q "$service")

        if [ -z "$cid" ]; then
            echo "$service: container not found" >&2
            exit 1
        fi

        state=$(docker inspect \
            --format '{{.State.Status}}' "$cid")

        health=$(docker inspect \
            --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' \
            "$cid")

        echo "$service: state=$state health=$health"

        case "$state" in
            exited|dead)
                echo "$service exited during startup" >&2
                docker compose logs --no-color "$service" >&2 || true
                exit 1
                ;;
        esac

        case "$health" in
            healthy|none)
                ;;
            *)
                all_healthy=false
                ;;
        esac
    done

    if [ "$all_healthy" = true ]; then
        break
    fi

    if [ "$(date +%s)" -ge "$deadline" ]; then
        echo "Timed out waiting for the Compose environment" >&2
        docker compose ps >&2 || true
        docker compose logs --no-color >&2 || true
        exit 1
    fi

    sleep 2
done

docker compose run --rm client sh -lc '
  set -eu
  conduit cp /mnt/fs_1/foo/hello.txt /mnt/fs_2/bar/
  conduit status
  test "$(cat /mnt/fs_2/bar/hello.txt)" = "hello"
'
