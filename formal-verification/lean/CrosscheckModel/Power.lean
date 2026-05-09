/-
Module: Power
Source informal spec: formal-verification/specs/power_informal.md
Source implementation: formal-verification/tests/power/sut/power.py
Sign-off: 2026-05-08
Pipeline step: 3 of 5 (/lean-impl). Next: /correspondence-review (3b.5).
-/

import Mathlib.Data.Nat.Basic
import Mathlib.Tactic.Linarith

namespace CrosscheckModel.Power

-- == Types =================================================================

-- Nat × Nat input domain; Nat output. No record type needed.

-- == Signatures ============================================================

-- src: formal-verification/tests/power/sut/power.py:1-12
/-- Natural-number exponentiation. Recurses on the exponent. -/
def power (base : Nat) (exp : Nat) : Nat :=
  match exp with
  | 0 => 1
  | n + 1 => base * power base n

-- == Properties ============================================================

/-- Q1: `power b 0 = 1` for every `b : Nat`. -/
theorem power_zero (b : Nat) : power b 0 = 1 := by sorry

/-- Q2: `power b (n+1) = b * power b n`. -/
theorem power_succ (b n : Nat) : power b (n + 1) = b * power b n := by sorry

/-- I2: `power 1 n = 1` for every `n : Nat`. -/
theorem power_one_base (n : Nat) : power 1 n = 1 := by sorry

/-- I3: `power b 1 = b` for every `b : Nat`. -/
theorem power_one_exp (b : Nat) : power b 1 = b := by sorry

end CrosscheckModel.Power
