-- Placeholder. lean-runner.sh overwrites this file at runtime with the
-- user's program. The placeholder imports a small Mathlib slice so the
-- image bake-step references real oleans (which keeps the cache warm for
-- common imports the user's spec stubs will reach for).
import Mathlib.Data.Nat.Defs
import Mathlib.Data.List.Basic
import Mathlib.Tactic.Linarith

namespace Crosscheck.Program

/-- Trivial placeholder so the harness module is non-empty at bake time. -/
def harnessReady : Bool := true

end Crosscheck.Program
