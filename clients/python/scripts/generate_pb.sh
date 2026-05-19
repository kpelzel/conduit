#!/usr/bin/env bash
set -euo pipefail

CLIENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$CLIENT_DIR/../.." && pwd)"

VIRTUAL_PROTO_ROOT="$CLIENT_DIR/build/proto"
VIRTUAL_PROTO_DIR="$VIRTUAL_PROTO_ROOT/conduit/_generated"
OUT_DIR="$CLIENT_DIR/src"

rm -rf "$VIRTUAL_PROTO_ROOT"
mkdir -p "$VIRTUAL_PROTO_DIR"
mkdir -p "$OUT_DIR/conduit/_generated"

cp "$REPO_ROOT/api/api.proto" "$VIRTUAL_PROTO_DIR/api.proto"

python3 -m grpc_tools.protoc \
  -I "$VIRTUAL_PROTO_ROOT" \
  --python_out="$OUT_DIR" \
  --grpc_python_out="$OUT_DIR" \
  --pyi_out="$OUT_DIR" \
  "$VIRTUAL_PROTO_DIR/api.proto"

touch "$OUT_DIR/conduit/_generated/__init__.py"