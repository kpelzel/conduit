#!/bin/sh
set -eu

MCP_DOCKER_PATH="/etc/conduit-mcp"
SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" >/dev/null 2>&1 && pwd)

LITELLM_URL="${LITELLM_URL:-https://litellm.home.arpa}"
LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY:-sk-litellm-mcp-key}"
STATE_FILE="$MCP_DOCKER_PATH/generated/litellm-testuser-key.env"
CADDY_ROOT_CA="${CADDY_ROOT_CA:-$MCP_DOCKER_PATH/caddy/caddy_data/caddy/pki/authorities/local/root.crt}"

TESTUSER_ID="testuser"
TESTUSER_EMAIL="testuser@example.com"

if [ -f "$STATE_FILE" ]; then
	echo "LiteLLM testuser key already exists: $STATE_FILE"
	exit 0
fi

need() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "Missing required command: $1" >&2
		exit 1
	}
}

need curl
need jq
need mktemp

mkdir -p "$MCP_DOCKER_PATH/generated"

curl_with_ca() {
	if [ -s "$CADDY_ROOT_CA" ]; then
		curl --cacert "$CADDY_ROOT_CA" "$@"
	else
		curl "$@"
	fi
}

litellm_post() {
	path="$1"
	body="$2"

	tmp_body="$(mktemp)"
	http_code=""

	if ! http_code="$(curl_with_ca -sS -o "$tmp_body" -w "%{http_code}" \
		-X POST "${LITELLM_URL}${path}" \
		-H "Authorization: Bearer ${LITELLM_MASTER_KEY}" \
		-H "Content-Type: application/json" \
		-d "$body")"; then
		echo "LiteLLM API request failed: POST ${LITELLM_URL}${path}" >&2
		cat "$tmp_body" >&2 || true
		rm -f "$tmp_body"
		exit 1
	fi

	case "$http_code" in
		2*)
			cat "$tmp_body"
			rm -f "$tmp_body"
			;;
		*)
			echo "LiteLLM API returned HTTP $http_code: POST ${LITELLM_URL}${path}" >&2
			echo "Request body:" >&2
			printf '%s\n' "$body" >&2
			echo "Response body:" >&2
			cat "$tmp_body" >&2 || true
			rm -f "$tmp_body"
			exit 1
			;;
	esac
}

wait_for_litellm() {
	i=0
	while ! curl_with_ca -fsS "${LITELLM_URL}/health/liveliness" >/dev/null 2>&1; do
		i=$((i + 1))
		if [ "$i" -gt 120 ]; then
			echo "Timed out waiting for LiteLLM at ${LITELLM_URL}" >&2
			exit 1
		fi
		sleep 1
	done
}

wait_for_litellm

echo "Creating LiteLLM virtual key for ${TESTUSER_ID}..."

BODY="$(jq -n \
	--arg user_id "$TESTUSER_ID" \
	--arg user_email "$TESTUSER_EMAIL" \
	'{
		user_id: $user_id,
		models: [],
		metadata: {
			user: $user_id,
			user_email: $user_email,
			created_by: "bootstrap-litellm.sh",
			purpose: "openwebui-mcp-demo"
		}
	}')"

RESP="$(litellm_post "/key/generate" "$BODY")"
KEY="$(printf '%s' "$RESP" | jq -r '.key // .token // empty')"

if [ -z "$KEY" ]; then
	echo "Could not find key in LiteLLM response:" >&2
	printf '%s\n' "$RESP" >&2
	exit 1
fi

umask 077
cat > "$STATE_FILE" <<EOF
LITELLM_TESTUSER_ID=${TESTUSER_ID}
LITELLM_TESTUSER_EMAIL=${TESTUSER_EMAIL}
LITELLM_TESTUSER_KEY=${KEY}
EOF

chmod 600 "$STATE_FILE"

echo
echo "LiteLLM testuser key created."
echo "Use this in Open WebUI MCP headers:"
echo
echo "  x-litellm-api-key: Bearer ${KEY}"
echo
echo "State written to:"
echo "  $STATE_FILE"