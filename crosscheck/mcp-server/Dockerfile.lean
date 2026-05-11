FROM --platform=linux/amd64 ubuntu:22.04

# Lean 4 + Mathlib harness image.
#
# Strategy: install elan, define a tiny Lake project at /harness that depends
# on a pinned Mathlib, then run `lake exe cache get` + `lake build` so all
# Mathlib oleans are baked into the image. At runtime, lean-runner.sh swaps
# the user's program into Crosscheck/Program.lean and re-runs `lake build`,
# which is fast because Mathlib doesn't need to recompile.
#
# Pin both the Lean toolchain (lean-toolchain file) and the Mathlib commit
# (lakefile.lean) — bump them together. See scripts/build-lean-docker.sh for
# rebuild cadence.

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install elan (Lean version manager) without picking a default toolchain;
# the lean-toolchain file in /harness selects it.
RUN curl -sSfL https://raw.githubusercontent.com/leanprover/elan/master/elan-init.sh \
    | sh -s -- --default-toolchain none -y
ENV PATH="/root/.elan/bin:${PATH}"

WORKDIR /harness

# Copy harness files. lean-toolchain pins Lean; lakefile.lean pins Mathlib.
COPY lean-harness/lean-toolchain /harness/lean-toolchain
COPY lean-harness/lakefile.lean /harness/lakefile.lean
COPY lean-harness/Crosscheck.lean /harness/Crosscheck.lean
COPY lean-harness/Crosscheck /harness/Crosscheck
COPY lean-harness/lean-runner.sh /harness/lean-runner.sh
RUN chmod +x /harness/lean-runner.sh

# Pull deps, fetch Mathlib's olean cache, then build the placeholder so all
# Mathlib oleans referenced by the placeholder import are resident.
RUN lake update \
    && (lake exe cache get || echo "cache get failed; will compile from source") \
    && lake build Crosscheck

# /work is the volume mount where the MCP server places program.lean.
ENTRYPOINT ["/harness/lean-runner.sh"]
