#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
	echo "Please run as root." >&2
	exit 1
fi

CONDUIT_MCP_PATH="/etc/conduit-mcp"

SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" >/dev/null 2>&1 && pwd)
echo "SCRIPT_DIR:$SCRIPT_DIR"

. "${SCRIPT_DIR}/vars.sh"

cd "$SCRIPT_DIR"

echo "Stopping and removing containers..."
$DOCKER_COMPOSE stop -t 10 || true
$DOCKER_COMPOSE rm --force || true

echo "Removing generated runtime data, preserving Caddy CA/data..."

mkdir -p "$CONDUIT_MCP_PATH"

# Remove everything under /etc/conduit-mcp except the caddy directory.
find "$CONDUIT_MCP_PATH" -mindepth 1 -maxdepth 1 ! -name caddy -exec rm -rf -- {} +

# Recreate expected directories for next build/run.
mkdir -p "$CONDUIT_MCP_PATH/generated"
mkdir -p "$CONDUIT_MCP_PATH/zitadel/postgres"
mkdir -p "$CONDUIT_MCP_PATH/zitadel/bootstrap"

echo
echo "Destroy complete."
echo "Preserved:"
echo "  $CONDUIT_MCP_PATH/caddy"
echo
echo "Caddy root CA should still be:"
echo "  $CONDUIT_MCP_PATH/caddy/caddy_data/caddy/pki/authorities/local/root.crt"