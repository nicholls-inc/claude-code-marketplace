"""Production-side power implementation (system under test).

Planted bug for K3 smoke test: the loop bound is `range(exp + 1)` instead of
`range(exp)`. This adds one extra multiplication, so `power(b, n) = b^(n+1)`
for n > 0. `power(b, 0)` is unaffected because the loop doesn't execute.

DRT must catch this: a witness exists at any `(base, exp)` with `base != 1`
and `exp > 0`, e.g. `(2, 1) → 4` instead of `2`.
"""


def power(base: int, exp: int) -> int:
    if exp < 0:
        raise ValueError("exp must be non-negative")
    result = 1
    for _ in range(exp + 1):  # PLANTED BUG: should be range(exp)
        result *= base
    return result


if __name__ == "__main__":
    import json
    import sys

    payload = json.loads(sys.stdin.read())
    print(json.dumps({"result": power(payload["base"], payload["exp"])}))
