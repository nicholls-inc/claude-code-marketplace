## Adversarial Review

I've read the plan, the issue, and the current artifacts. Now I'll probe for failure modes.

---

### **1. Spec / intent gap** — Plan solves symptoms, not the root causes

The field report identifies three **critical failures** (W1, W2, W3):
- **W1**: `/trace-execution` missed a critical 5-line filter in the PRIMARY ENTRY POINT
- **W2**: Overconfident assertion after incomplete trace ("definitively possible" when a filter was missed)
- **W3**: `/reason` proposed fix reintroduced the TOCTOU pattern

The plan proposes **instruction clarifications** (adding abort conditions, strengthening guidance, adding cross-references). But:

- **W1 root cause per field report**: "The `/trace-execution` skill may lack explicit instruction to read ALL lines of each function in the path." The plan's Step 2 proposes "clarify when to continue tracing vs. when to conclude." **This doesn't address W1** — the issue isn't about *when to stop*, it's about **reading exhaustively** before deciding.

- **W3 root cause per field report**: "The alternative hypothesis check in `/reason` is oriented toward the *diagnosis* rather than the *prescription*." The plan proposes Step 1 "strengthen the requirement that premises MUST cite file:line locations" and Step 3 "add examples of what to reject." **Neither addresses W3** — the skill already has file:line citations; it just doesn't verify the *proposed fix*.

- **W2 root cause**: No confidence calibration guardrails. The plan doesn't address this at all.

The field report explicitly recommends **R1** (add "verify the fix" step to `/reason`) and **R2** (add "exhaustive function read" instruction to `/trace-execution`). The plan doesn't implement either.

**Verdict**: The plan addresses peripheral issues (abort conditions, mandatory rationales) but **misses the three P0 fixes** the field report called for.

---

### **2. Hidden assumptions** — "Refactor" assumes instruction changes solve agent behavior

**Plan Section 3 (Step 4)** assumes "cross-references between skills and orchestrator" will improve outcomes. But:
- The field report notes: "Skills loaded at runtime, not in initial context" (W6).
- Cross-references in SKILL.md files won't change byfuglien's approach decision if byfuglien doesn't have the skill methodology loaded when it classifies tasks.

The plan doesn't address W6, yet assumes cross-linking will work. This is circular — if the skill isn't loaded, the cross-reference isn't read.

---

### **3. Missing edge cases** — No test plan for the refactor

**Plan Section: "Tests / properties to add"** proposes:
1. "Instruction completeness check" — checking that abort conditions exist
2. "Consistency verification" — quality gates match evidence formats
3. "Workflow coherence trace" — trace a hypothetical invocation

But:
- **None of these would detect W1** (missing a filter in actual code). The "hypothetical invocation" isn't a real code scenario with a real 5-line filter to miss.
- **None would detect W3** (proposed fix reintroducing TOCTOU). The "workflow coherence" just checks if the agent produces output that passes validation — it doesn't check if the *fix* is correct.

The test plan is tautological: it checks that instructions exist and are self-consistent, not that they **prevent the specific failures** documented in the field report.

---

### **4. Test adequacy** — Would proposed tests fail if W1/W3 recurred?

No. The plan's verification approach (Section "Tests / properties to add") would pass even with the current versions of the skills that produced W1/W3.

The field report says: "The user was the best crosschecker. In this session, the user caught two flaws that byfuglien missed (W1: filter, W3: fix reintroduces TOCTOU)."

The plan doesn't add any mechanical check to catch these. It only adds **instruction clarifications** that may or may not change agent behavior.

---

### **5. Scope creep** — Step 4 adds cross-references not required by the issue

**Plan Step 4**: "Add cross-references between skills and orchestrator."

The field report does NOT recommend this. The field report's P0 recommendations are:
- R1: Add "verify the fix" step to `/reason`
- R2: Add "exhaustive function read" instruction to `/trace-execution`

The plan's Step 4 is speculative ("ensure skills reference their validation by byfuglien"). There's no evidence in the field report that this would address W1, W2, or W3.

---

### **6. Verification track fit** — "Semi-formal" track claimed but not applied to the plan itself

The plan claims **semi-formal verification** and includes a "Verification approach" section with premises P1-P3 and conclusions C1-C3.

But:
- **P2**: "The current `/reason` and `/trace-execution` skills have evidence requirements but unclear abort thresholds."
  - **Not evidence-grounded** — the plan doesn't cite specific line numbers in the SKILL.md files showing where abort thresholds are unclear.
  
- **C1**: "Strengthening abort conditions in skills prevents premature conclusions."
  - **No evidence** that W1 (missed filter) was due to unclear abort conditions. The field report says the skill may have skimmed rather than reading exhaustively — that's not an abort condition problem.

The plan's own "semi-formal reasoning" doesn't meet the standards it's supposed to enforce.

---

### **7. Alternative hypothesis check** — Plan doesn't consider that instruction changes may not fix agent behavior

**Plan Section: "Alternative hypothesis check"** considers: "The performance issues are due to model limitations, not instruction clarity."

But it doesn't check: **"Will clarifying instructions actually prevent W1/W3?"**

The field report notes: "The `/trace-execution` skill appears to have skimmed the entry function rather than reading every line." This could be:
1. Missing instruction (plan's hypothesis)
2. Instruction present but ignored by the model
3. Instruction present but agent prioritized speed over thoroughness

The plan assumes (1) without checking (2) or (3). If the current SKILL.md already says "read the code" but the agent skimmed anyway, adding "read ALL lines" may not help.

---

## Concerns for Revision

1. **Plan doesn't implement the field report's P0 recommendations (R1, R2).** The plan proposes instruction clarifications around abort conditions and rationales, but the field report explicitly says: "Add 'verify the fix' step to `/reason`" (R1) and "Add 'exhaustive function read' instruction to `/trace-execution`" (R2). Neither is in the plan.

2. **Plan's verification approach violates its own evidence standards.** Premise P2 claims "unclear abort thresholds" without citing specific SKILL.md line numbers. Claim C1 asserts a causal relationship without evidence that unclear abort conditions caused W1.

3. **Test plan is tautological.** The "instruction completeness check" and "workflow coherence trace" would pass with the current SKILL.md files that produced W1/W3. No proposed test would fail if the documented bugs recurred.

4. **Scope creep in Step 4.** Cross-referencing skills and orchestrator is not mentioned in the field report's recommendations and there's no evidence it addresses W1, W2, or W3.

5. **Hidden assumption about agent behavior.** Plan assumes clarifying instructions will change agent behavior, but doesn't address the possibility that current instructions are ignored or deprioritized. Field report notes skills are loaded at runtime (W6) and agent may have skimmed despite instructions.

6. **Missing W2 fix entirely.** Field report R3 recommends adding confidence calibration guardrails to `/trace-execution`. Plan doesn't address this.

```json
{
  "preferred_next_label": "revise",
  "context_updates": {
    "review_verdict": "revise",
    "review_concerns": "1. Plan doesn't implement field report's P0 recommendations R1 (add 'verify the fix' step to /reason) and R2 (add 'exhaustive function read' instruction to /trace-execution). Current steps focus on abort conditions and cross-references, not the specific failures documented.\n2. Plan's own semi-formal verification violates evidence standards — P2 claims 'unclear abort thresholds' without citing SKILL.md line numbers; C1 asserts causality without evidence.\n3. Test plan is tautological — proposed checks (instruction completeness, workflow coherence) would pass with current SKILL.md files that produced W1/W3. No mechanical test to catch the documented failures.\n4. Scope creep in Step 4 — cross-referencing not in field report recommendations, no evidence it addresses W1/W2/W3.\n5. Hidden assumption that instruction changes fix agent behavior, but field report notes skills loaded at runtime (W6) and agent may have skimmed despite instructions.\n6. Missing W2 fix — field report R3 recommends confidence calibration guardrails for /trace-execution; plan doesn't address this."
  }
}
```