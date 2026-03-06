#!/usr/bin/env bash
set -euo pipefail

# Smoke test for the crosscheck MCP server.
# Prerequisites: npm run build in mcp-server/, Docker image built.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER="node ${SCRIPT_DIR}/../mcp-server/dist/index.js"

# Helper: send JSON-RPC request and capture response
send_request() {
  local request="$1"
  echo "${request}" | DAFNY_DOCKER_IMAGE="${DAFNY_DOCKER_IMAGE:-crosscheck-dafny:latest}" ${SERVER} 2>/dev/null
}

echo "=== Test 1: Initialize ==="
INIT_REQ='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}'
INIT_RESP=$(send_request "${INIT_REQ}")
echo "Init response: ${INIT_RESP}"

echo ""
echo "=== Test 2: Verify valid Dafny ==="
VALID_SOURCE='method Max(a: int, b: int) returns (r: int)\n  ensures r >= a && r >= b\n  ensures r == a || r == b\n{\n  if a >= b { r := a; } else { r := b; }\n}'
VERIFY_REQ="{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"dafny_verify\",\"arguments\":{\"source\":\"${VALID_SOURCE}\"}}}"
echo "Sending verify request for valid Dafny..."
VERIFY_RESP=$(send_request "${VERIFY_REQ}")
echo "Verify response: ${VERIFY_RESP}"

echo ""
echo "=== Test 3: Verify invalid Dafny ==="
INVALID_SOURCE='method Bad() returns (r: int)\n  ensures r > 0\n{\n  r := -1;\n}'
VERIFY_BAD_REQ="{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"dafny_verify\",\"arguments\":{\"source\":\"${INVALID_SOURCE}\"}}}"
echo "Sending verify request for invalid Dafny..."
VERIFY_BAD_RESP=$(send_request "${VERIFY_BAD_REQ}")
echo "Verify response (should have errors): ${VERIFY_BAD_RESP}"

echo ""
echo "=== Test 4: Cleanup ==="
CLEANUP_REQ='{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"dafny_cleanup","arguments":{}}}'
CLEANUP_RESP=$(send_request "${CLEANUP_REQ}")
echo "Cleanup response: ${CLEANUP_RESP}"

echo ""
echo "=== All smoke tests completed ==="
