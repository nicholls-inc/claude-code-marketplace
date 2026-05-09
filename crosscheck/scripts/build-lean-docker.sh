#!/usr/bin/env bash
set -euo pipefail

# Build the Lean + Mathlib harness Docker image used by the MCP server's
# lean_check / lean_run / lean_test tools.
#
# REBUILD CADENCE
#   - When `mcp-server/lean-harness/lean-toolchain` is bumped (Lean release).
#   - When `mcp-server/lean-harness/lakefile.lean` Mathlib pin is bumped.
#   - Periodically (~quarterly) to pick up upstream security fixes in the
#     Ubuntu base layer.
#
# The first build is slow (Mathlib compile / cache fetch). Subsequent builds
# are fast as long as the lean-toolchain and Mathlib pins are stable.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKERFILE="${SCRIPT_DIR}/../mcp-server/Dockerfile.lean"
CONTEXT="${SCRIPT_DIR}/../mcp-server"
IMAGE_NAME="${LEAN_DOCKER_IMAGE:-crosscheck-lean:latest}"

echo "Building Lean+Mathlib Docker image: ${IMAGE_NAME}"
echo "First-time builds may take 10-30 minutes (Mathlib oleans). Subsequent builds reuse layers."
docker build -t "${IMAGE_NAME}" -f "${DOCKERFILE}" "${CONTEXT}"

echo "Verifying image..."
docker run --rm "${IMAGE_NAME}" check /dev/null 2>&1 \
  || echo "(harness check returned non-zero on /dev/null — expected; the image is ready)"

# Pre-warming sanity timing (byfuglien C-B1). Runs `lean_check` against a tiny
# file with a single Mathlib import. With the image freshly built and Mathlib
# oleans baked in, this should complete in <10 seconds. If it takes much
# longer, the pre-warming claim has regressed — investigate before relying on
# the 240s default timeout to mask the problem.
TIMING_DIR=$(mktemp -d)
trap 'rm -rf "${TIMING_DIR}"' EXIT
cat > "${TIMING_DIR}/program.lean" <<'EOF'
import Mathlib.Data.Nat.Defs
example : 1 + 1 = 2 := rfl
EOF
echo "Timing baseline: lake build on a 2-line Mathlib-importing file..."
SECONDS=0
docker run --rm --network=none -v "${TIMING_DIR}:/work" "${IMAGE_NAME}" \
  check /work/program.lean >/dev/null 2>&1 || true
echo "  elapsed: ${SECONDS}s (expected: <10s with pre-warmed Mathlib)"

# Pin-drift advisory (byfuglien C-B2). The lakefile.lean inside the image
# pins Mathlib at build time. If a downstream user edits the host-side
# lakefile.lean and does not re-run this script, runtime builds will fall
# back to from-source Mathlib compilation. There is no automated drift
# detection — bump the pin and rebuild together.

echo "Done. Image: ${IMAGE_NAME}"
