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

echo "Removing generated runtime data, preserving Caddy CA and Open WebUI model cache..."

mkdir -p "$CONDUIT_MCP_PATH"

# Preserve Open WebUI model cache if it exists
OPENWEBUI_MODEL_CACHE="$CONDUIT_MCP_PATH/openwebui/cache/embedding/models"
TEMP_MODEL_CACHE="/tmp/conduit-mcp-model-cache-$$"
if [ -d "$OPENWEBUI_MODEL_CACHE" ]; then
	echo "Backing up Open WebUI model cache..."
	mkdir -p "$TEMP_MODEL_CACHE"
	cp -a "$OPENWEBUI_MODEL_CACHE" "$TEMP_MODEL_CACHE/"
fi

# Remove everything under /etc/conduit-mcp except the caddy directory.
find "$CONDUIT_MCP_PATH" -mindepth 1 -maxdepth 1 ! -name caddy -exec rm -rf -- {} +

# Recreate expected directories for next build/run.
mkdir -p "$CONDUIT_MCP_PATH/generated"
mkdir -p "$CONDUIT_MCP_PATH/zitadel/postgres"
mkdir -p "$CONDUIT_MCP_PATH/zitadel/bootstrap"

# Restore model cache if it was backed up
if [ -d "$TEMP_MODEL_CACHE/models" ]; then
	echo "Restoring Open WebUI model cache..."
	mkdir -p "$OPENWEBUI_MODEL_CACHE"
	cp -a "$TEMP_MODEL_CACHE/models" "$CONDUIT_MCP_PATH/openwebui/cache/embedding/"
	rm -rf "$TEMP_MODEL_CACHE"
fi

echo
echo "Destroy complete."
echo "Preserved:"
echo "  $CONDUIT_MCP_PATH/caddy (CA certificates)"
if [ -d "$OPENWEBUI_MODEL_CACHE" ]; then
	echo "  $CONDUIT_MCP_PATH/openwebui/cache/embedding/models (model cache)"
fi
echo
echo "Caddy root CA should still be:"
echo "  $CONDUIT_MCP_PATH/caddy/caddy_data/caddy/pki/authorities/local/root.crt"
