# Spec Registry Schema — `.crosscheck/specs.json`

The spec registry is a JSON manifest at the project root that tracks verified Dafny specifications and their associated extracted code. It enables regression detection by recording what was verified, when, and where the extracted code lives.

## Schema

```json
{
  "version": 1,
  "specs": [
    {
      "id": "unique-slug",
      "function": "DafnyMethodName",
      "description": "Natural-language description of what the spec verifies",
      "dafnySource": "relative/path/to/spec.dfy",
      "dafnySourceHash": "sha256:abc123...",
      "extractedCode": {
        "file": "relative/path/to/output.py",
        "function": "python_function_name",
        "language": "python"
      },
      "constraint": "hard",
      "lastVerified": "2026-03-12T14:30:00Z",
      "difficulty": {
        "solverTimeMs": 1200,
        "resourceCount": 45000,
        "proofHintCount": 2,
        "trivialProof": false
      },
      "trustBoundaries": [
        "No IO verification — file reads are trust boundaries",
        "float precision not guaranteed (uses Dafny real type)"
      ]
    }
  ]
}
```

## Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | number | yes | Schema version, currently `1` |
| `specs` | array | yes | Array of spec entries |
| `specs[].id` | string | yes | Unique identifier slug (e.g., `max-of-array`, `split-energy`) |
| `specs[].function` | string | yes | Dafny method/function name as it appears in the `.dfy` source |
| `specs[].description` | string | yes | Human-readable description of the verified property |
| `specs[].dafnySource` | string | yes | Relative path from project root to the `.dfy` file |
| `specs[].dafnySourceHash` | string | yes | `sha256:` prefixed hash of the Dafny source file contents |
| `specs[].extractedCode.file` | string | yes | Relative path from project root to the extracted code file |
| `specs[].extractedCode.function` | string | yes | Function name in the extracted code |
| `specs[].extractedCode.language` | string | yes | `"python"` or `"go"` |
| `specs[].constraint` | string | yes | `"hard"` (must pass `dafny_verify`) or `"soft"` (property-based tests suffice) |
| `specs[].lastVerified` | string | yes | ISO 8601 timestamp of last successful verification |
| `specs[].difficulty` | object | no | Verification difficulty metrics from the last run |
| `specs[].difficulty.solverTimeMs` | number | no | Z3 solver time in milliseconds |
| `specs[].difficulty.resourceCount` | number | no | Dafny resource count |
| `specs[].difficulty.proofHintCount` | number | no | Number of manual proof hints (assert, calc, etc.) |
| `specs[].difficulty.trivialProof` | boolean | no | Whether the proof was trivially discharged |
| `specs[].trustBoundaries` | string[] | no | List of known limitations and trust assumptions |

## Notes

- All paths are relative to the project root (where `.crosscheck/` lives)
- The `dafnySourceHash` is used to detect whether the spec itself has changed (not just the extracted code)
- `constraint: "soft"` entries are verified via property-based tests rather than full Dafny re-verification during `/check-regressions`
- The registry is created by `/extract-code` (Step 5.5) and consumed by `/check-regressions`
- Users can manually edit the registry (e.g., to change constraint strength or remove stale entries)
