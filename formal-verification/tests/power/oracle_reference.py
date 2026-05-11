"""Lean-faithful reference oracle (used while the Lean image is not yet built).

Mirrors the recursion of `CrosscheckModel.Power.power` exactly:
    power(b, 0) = 1
    power(b, n+1) = b * power(b, n)

This is the *smoke-time* oracle. Once the Lean image builds, the harness
should switch to invoking the Lean runner via `lake exe PowerRunner` (or via
the MCP tool `lean_run`). Until then, this Python reference plays the oracle
role and the smoke is honest about that scope.
"""


def power(base: int, exp: int) -> int:
    if exp == 0:
        return 1
    return base * power(base, exp - 1)


if __name__ == "__main__":
    import json
    import sys

    payload = json.loads(sys.stdin.read())
    print(json.dumps({"result": power(payload["base"], payload["exp"])}))
