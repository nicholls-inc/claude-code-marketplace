# Gate: Lowry Drift Packet

## What this gate protects

`lowry` is Crosscheck's implementation-loop agent: it drives code from a failing ("red") build to a passing ("green") one against a set of already-approved correctness rules, called **invariants**. Its one hard rule is that it will never reach green by weakening, deleting, or silently editing an invariant. If the only way forward would do that, `lowry` stops and writes a **drift packet** instead: a staged `governance-amendment` commit that asks the canonical question — did we actually want this behaviour, or did the implementation just drift away from what was agreed?

This gate exists so that a build can never look "done" by quietly loosening the very rules it was supposed to satisfy. A green build reached that way would be false confidence, not real progress.

## What you are being asked to decide

`lowry` has hit a point where continuing straight to green requires weakening an invariant, and it has refused to do so on its own. It has written a drift packet — a proposed commit describing the invariant in question and its justification for why the behaviour might need to change. You are being asked to resolve that packet: either accept it as a legitimate, deliberate change to the invariant, or decide the implementation itself has drifted and needs to be corrected instead.

There is no silent third option. Crosscheck treats only two outcomes as valid endings for this loop: **green without weakening anything**, or **stopped with a drift packet awaiting your decision**. `lowry` will not proceed past this point, and it will not quietly abandon the attempt, until you decide.

## Your options

- **Accept the staged governance amendment.** This confirms the invariant itself should change to reflect a genuine, agreed shift in intended behaviour. The amendment commit lands, and `lowry` resumes the loop against the updated invariant.
- **Send the implementation back.** This confirms the invariant was correct as written and the code drifted from it. The amendment commit is discarded, and the implementation is reworked to satisfy the original invariant rather than to change it.

## What happens next in each case

- **Accepted:** the governance-amendment commit is kept, `lowry` resumes driving to green against the amended contract, and the change still goes through Crosscheck's normal review and intent-check steps before merge — accepting the amendment is not the same as approving the final result.
- **Sent back:** the drift packet is discarded, no invariant changes, and `lowry` continues the run-to-green loop against the invariants as originally approved.

## How long this takes

Usually a few minutes — the packet is a single, focused justification for one invariant, not a batch of unrelated changes.

## Who to ask if unsure

The Crosscheck maintainers, via a GitHub issue on this repository.
