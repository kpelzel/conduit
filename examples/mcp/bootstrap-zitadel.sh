#!/bin/sh
set -eu

MCP_DOCKER_PATH="/etc/conduit-mcp"
SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" >/dev/null 2>&1 && pwd)

ZITADEL_URL="${ZITADEL_URL:-https://zitadel.home.arpa}"
API="${ZITADEL_URL}/management/v1"
PAT_FILE="$MCP_DOCKER_PATH/zitadel/bootstrap/admin.pat"
STATE_FILE="$MCP_DOCKER_PATH/generated/zitadel-bootstrap-state.env"
CADDY_ROOT_CA="${CADDY_ROOT_CA:-$MCP_DOCKER_PATH/caddy/caddy_data/caddy/pki/authorities/local/root.crt}"

DEMO_USERNAME="testuser"
DEMO_EMAIL="testuser@example.com"
DEMO_PASSWORD="password"

if [ -f "$STATE_FILE" ]; then
	echo "ZITADEL already bootstrapped: $STATE_FILE"
	exit 0
fi

if [ ! -s "$PAT_FILE" ]; then
	echo "Missing ZITADEL bootstrap PAT at $PAT_FILE" >&2
	exit 1
fi

PAT="$(cat "$PAT_FILE")"

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

api_request_url() {
	method="$1"
	url="$2"
	body="$3"

	tmp_body="$(mktemp)"
	http_code=""

	if ! http_code="$(curl_with_ca -sS -o "$tmp_body" -w "%{http_code}" \
		-X "$method" "$url" \
		-H "Authorization: Bearer ${PAT}" \
		-H "Content-Type: application/json" \
		-d "$body")"; then
		echo "ZITADEL API request failed: $method $url" >&2
		echo "Response body:" >&2
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
			echo "ZITADEL API returned HTTP $http_code: $method $url" >&2
			echo "Request body:" >&2
			printf '%s\n' "$body" >&2
			echo "Response body:" >&2
			cat "$tmp_body" >&2 || true
			rm -f "$tmp_body"
			exit 1
			;;
	esac
}

api_request_url_optional() {
	method="$1"
	url="$2"
	body="$3"

	tmp_body="$(mktemp)"
	http_code=""

	if ! http_code="$(curl_with_ca -sS -o "$tmp_body" -w "%{http_code}" \
		-X "$method" "$url" \
		-H "Authorization: Bearer ${PAT}" \
		-H "Content-Type: application/json" \
		-d "$body")"; then
		rm -f "$tmp_body"
		return 1
	fi

	case "$http_code" in
		2*)
			cat "$tmp_body"
			rm -f "$tmp_body"
			return 0
			;;
		*)
			echo "Optional API call failed with HTTP $http_code: $method $url" >&2
			echo "Response body:" >&2
			cat "$tmp_body" >&2 || true
			rm -f "$tmp_body"
			return 1
			;;
	esac
}

api_post() {
	path="$1"
	body="$2"
	api_request_url "POST" "${API}${path}" "$body"
}

relax_demo_password_policy() {
	body="$(jq -n '{
	  minLength: 8,
	  hasUppercase: false,
	  hasLowercase: true,
	  hasNumber: false,
	  hasSymbol: false
	}')"

	echo "Relaxing demo password policy so testuser/password works..."

	if api_request_url_optional "PUT" "${ZITADEL_URL}/policies/password/complexity" "$body" >/dev/null; then
		return 0
	fi

	api_request_url "PUT" "${ZITADEL_URL}/admin/v1/policies/password/complexity" "$body" >/dev/null
}

json_field() {
	json="$1"
	field="$2"
	value="$(printf '%s' "$json" | jq -r "$field")"

	if [ -z "$value" ] || [ "$value" = "null" ]; then
		echo "Expected JSON field missing: $field" >&2
		echo "JSON was:" >&2
		printf '%s\n' "$json" >&2
		exit 1
	fi

	printf '%s\n' "$value"
}

echo "Using ZITADEL URL: $ZITADEL_URL"
if [ -s "$CADDY_ROOT_CA" ]; then
	echo "Using Caddy root CA: $CADDY_ROOT_CA"
else
	echo "Caddy root CA not found at $CADDY_ROOT_CA; relying on system trust store."
fi

relax_demo_password_policy

echo "Creating ZITADEL project..."
PROJECT_BODY="$(jq -n '{
  name: "conduit-lab",
  projectRoleAssertion: true,
  projectRoleCheck: false,
  hasProjectCheck: false,
  privateLabelingSetting: "PRIVATE_LABELING_SETTING_UNSPECIFIED"
}')"
PROJECT_JSON="$(api_post "/projects" "$PROJECT_BODY")"
PROJECT_ID="$(json_field "$PROJECT_JSON" '.id')"

echo "Creating Open WebUI OIDC app..."
OPENWEBUI_BODY="$(jq -n '{
  name: "openwebui",
  redirectUris: ["https://openwebui.home.arpa/oauth/oidc/callback"],
  responseTypes: ["OIDC_RESPONSE_TYPE_CODE"],
  grantTypes: [
    "OIDC_GRANT_TYPE_AUTHORIZATION_CODE",
    "OIDC_GRANT_TYPE_REFRESH_TOKEN"
  ],
  appType: "OIDC_APP_TYPE_WEB",
  authMethodType: "OIDC_AUTH_METHOD_TYPE_BASIC",
  version: "OIDC_VERSION_1_0",
  devMode: false,
  accessTokenType: "OIDC_TOKEN_TYPE_BEARER",
  accessTokenRoleAssertion: true,
  idTokenRoleAssertion: true,
  idTokenUserinfoAssertion: true,
  clockSkew: "0s"
}')"
OPENWEBUI_JSON="$(api_post "/projects/${PROJECT_ID}/apps/oidc" "$OPENWEBUI_BODY")"
OPENWEBUI_CLIENT_ID="$(json_field "$OPENWEBUI_JSON" '.clientId')"
OPENWEBUI_CLIENT_SECRET="$(json_field "$OPENWEBUI_JSON" '.clientSecret')"

echo "Creating LiteLLM OBO OIDC app..."
LITELLM_BODY="$(jq -n '{
  name: "litellm-obo",
  redirectUris: ["https://litellm.home.arpa/unused-oidc-callback"],
  responseTypes: ["OIDC_RESPONSE_TYPE_CODE"],
  grantTypes: [
    "OIDC_GRANT_TYPE_AUTHORIZATION_CODE",
    "OIDC_GRANT_TYPE_REFRESH_TOKEN",
    "OIDC_GRANT_TYPE_TOKEN_EXCHANGE"
  ],
  appType: "OIDC_APP_TYPE_WEB",
  authMethodType: "OIDC_AUTH_METHOD_TYPE_BASIC",
  version: "OIDC_VERSION_1_0",
  devMode: false,
  accessTokenType: "OIDC_TOKEN_TYPE_BEARER",
  accessTokenRoleAssertion: true,
  idTokenRoleAssertion: true,
  idTokenUserinfoAssertion: true,
  clockSkew: "0s"
}')"
LITELLM_JSON="$(api_post "/projects/${PROJECT_ID}/apps/oidc" "$LITELLM_BODY")"
LITELLM_OBO_CLIENT_ID="$(json_field "$LITELLM_JSON" '.clientId')"
LITELLM_OBO_CLIENT_SECRET="$(json_field "$LITELLM_JSON" '.clientSecret')"

echo "Creating Conduit MCP API app..."
CONDUIT_BODY="$(jq -n '{
  name: "conduit-mcp",
  authMethodType: "API_AUTH_METHOD_TYPE_BASIC"
}')"
CONDUIT_JSON="$(api_post "/projects/${PROJECT_ID}/apps/api" "$CONDUIT_BODY")"
CONDUIT_CLIENT_ID="$(json_field "$CONDUIT_JSON" '.clientId')"
CONDUIT_CLIENT_SECRET="$(json_field "$CONDUIT_JSON" '.clientSecret')"

echo "Creating demo user ${DEMO_USERNAME}..."
USER_BODY="$(jq -n \
  --arg username "$DEMO_USERNAME" \
  --arg email "$DEMO_EMAIL" \
  --arg password "$DEMO_PASSWORD" \
  '{
    userName: $username,
    profile: {
      firstName: "Test",
      lastName: "User",
      nickName: "testuser",
      displayName: "Test User",
      preferredLanguage: "en",
      gender: "GENDER_UNSPECIFIED"
    },
    email: {
      email: $email,
      isEmailVerified: true
    },
    initialPassword: $password
  }')"
USER_JSON="$(api_post "/users/human" "$USER_BODY")"
DEMO_USER_ID="$(json_field "$USER_JSON" '.userId')"

AUD_SCOPE="urn:zitadel:iam:org:project:id:${PROJECT_ID}:aud"

umask 077

cat > "$MCP_DOCKER_PATH/generated/openwebui.env" <<EOF
ENABLE_OAUTH_SIGNUP=true
OAUTH_CLIENT_ID=${OPENWEBUI_CLIENT_ID}
OAUTH_CLIENT_SECRET=${OPENWEBUI_CLIENT_SECRET}
OAUTH_PROVIDER_NAME=ZITADEL
OPENID_PROVIDER_URL=https://zitadel.home.arpa/.well-known/openid-configuration
OPENID_REDIRECT_URI=https://openwebui.home.arpa/oauth/oidc/callback
OAUTH_MERGE_ACCOUNTS_BY_EMAIL=true
OAUTH_SCOPES=openid email profile offline_access ${AUD_SCOPE}
EOF

cat > "$MCP_DOCKER_PATH/generated/litellm.env" <<EOF
ZITADEL_PROJECT_ID=${PROJECT_ID}
ZITADEL_LITELLM_OBO_CLIENT_ID=${LITELLM_OBO_CLIENT_ID}
ZITADEL_LITELLM_OBO_CLIENT_SECRET=${LITELLM_OBO_CLIENT_SECRET}
EOF

cat > "$MCP_DOCKER_PATH/generated/conduit-mcp.env" <<EOF
CONDUIT_MCP_OAUTH_CLIENT_ID=${CONDUIT_CLIENT_ID}
CONDUIT_MCP_OAUTH_CLIENT_SECRET=${CONDUIT_CLIENT_SECRET}
CONDUIT_MCP_OAUTH_ISSUER=https://zitadel.home.arpa
CONDUIT_MCP_OAUTH_INTROSPECTION_URL=https://zitadel.home.arpa/oauth/v2/introspect
CONDUIT_MCP_OAUTH_USERINFO_URL=https://zitadel.home.arpa/oidc/v1/userinfo
EOF

cat > "$SCRIPT_DIR/config_files/litellm-config.yaml" <<EOF
model_list: []

general_settings:
  master_key: os.environ/LITELLM_MASTER_KEY

mcp_servers:
  conduit_mcp:
    url: "https://mcp.home.arpa/mcp"
    transport: "http"
    auth_type: oauth2_token_exchange

    token_exchange_endpoint: "https://zitadel.home.arpa/oauth/v2/token"
    client_id: os.environ/ZITADEL_LITELLM_OBO_CLIENT_ID
    client_secret: os.environ/ZITADEL_LITELLM_OBO_CLIENT_SECRET

    audience: "${PROJECT_ID}"
    scopes:
      - "openid"
      - "profile"
      - "email"

    subject_token_type: "urn:ietf:params:oauth:token-type:access_token"
EOF

cat > "$STATE_FILE" <<EOF
ZITADEL_PROJECT_ID=${PROJECT_ID}
ZITADEL_AUDIENCE_SCOPE=${AUD_SCOPE}
OPENWEBUI_CLIENT_ID=${OPENWEBUI_CLIENT_ID}
LITELLM_OBO_CLIENT_ID=${LITELLM_OBO_CLIENT_ID}
CONDUIT_CLIENT_ID=${CONDUIT_CLIENT_ID}
DEMO_USER_ID=${DEMO_USER_ID}
DEMO_USERNAME=${DEMO_USERNAME}
DEMO_EMAIL=${DEMO_EMAIL}
DEMO_PASSWORD=${DEMO_PASSWORD}
EOF

chmod 600 "$MCP_DOCKER_PATH/generated/"*.env "$STATE_FILE"
chmod 600 "$SCRIPT_DIR/config_files/litellm-config.yaml"

echo
echo "ZITADEL bootstrap complete."
echo "Demo login:"
echo "  username: ${DEMO_USERNAME}"
echo "  password: ${DEMO_PASSWORD}"
echo
echo "State written to:"
echo "  $STATE_FILE"