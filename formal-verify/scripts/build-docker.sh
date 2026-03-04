#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKERFILE="${SCRIPT_DIR}/../mcp-server/Dockerfile"
IMAGE_NAME="${DAFNY_DOCKER_IMAGE:-formal-verify-dafny:latest}"

echo "Building Dafny Docker image: ${IMAGE_NAME}"
docker build -t "${IMAGE_NAME}" -f "${DOCKERFILE}" "${SCRIPT_DIR}/../mcp-server"

echo "Verifying image..."
docker run --rm "${IMAGE_NAME}" --version

echo "Done. Image: ${IMAGE_NAME}"
