import Lake
open Lake DSL

require mathlib from git
  "https://github.com/leanprover-community/mathlib4.git" @ "v4.10.0"

package crosscheck where

@[default_target]
lean_lib Crosscheck where
  -- Crosscheck.lean is the root; it imports Crosscheck.Program (the swapped
  -- user file). Keep the surface minimal so re-builds at runtime are cheap.
  globs := #[.submodules `Crosscheck]
