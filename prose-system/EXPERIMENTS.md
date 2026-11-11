# Experiments

Eight experiments are queued. None has been run. Experiments 1 to 3 need no human readers and can run as soon as the harness and the brief sets exist. Experiments 4 to 8 need the reader protocol in `EVALUATION.md`.

## Rules that apply to every experiment

- Every sample comes from its own API call with a fresh context. The call contains the condition's prompt and nothing else. No sample is written by an agent that has read `BRIEF.md`, `FINDINGS.md` or this file.
- At least 10 samples per brief per condition. Temperature and other sampling settings are left at the API defaults unless a condition says otherwise, and they are recorded.
- Each stored sample carries the model identifier, the full prompt, the sampling settings, the condition name, the brief identifier and a timestamp.
- Experiments 1 to 3 and all tuning use the 12 development briefs. The 24 held-out briefs are used only to confirm a result that will change the system.
- The profile script is `scripts/profile.py`. Pool samples within a condition before reading the rhythm measures, because one document is too short for them.
- State the expected result before running, then report what happened. Report null results.

## 1. Baseline

Purpose: fix the comparison for everything else, and check whether the tells described in the cited studies appear in the current model.

- Condition: one plain prompt per brief, written the way a competent user would ask. No style instructions.
- Measures: the full profile, pooled and per sample. Where a human reference text exists for the brief, the same profile on that text.
- Output: stored samples that are never regenerated, and a table of baseline against human reference.

## 2. Substitution

Purpose: test the brief's first failure claim.

- Condition A: baseline.
- Condition B: baseline prompt with an instruction not to use em dashes.
- Condition C: baseline prompt with an instruction not to use dashes, semicolons, colons or parentheses.
- Measures: each mark per 1,000 words, and the combined rate of the four marks.
- Reading: the claim is supported if B removes dashes and the combined rate stays near A. Look at the samples to confirm the new marks sit where a dash would have gone. In C, look for what replaces the four marks, including sentence fragments and comma splices, and count it.

## 3. Positive rule

Purpose: test the brief's second failure claim.

- Condition A: baseline.
- Condition B: baseline prompt with an instruction to vary sentence length.
- Measures, pooled across samples: coefficient of variation of sentence length, lag-1 autocorrelation of sentence length, and the share of sentences of 8 words or fewer that are followed by a sentence at least 2.5 times longer.
- Reading: the claim is supported if B lowers the autocorrelation or raises the short-then-long share against A. Compare both with a human reference built with the same script.

## 4. Material

Purpose: test hypothesis 3, that missing observation drives the general claims readers penalise.

- Condition A: baseline.
- Condition B: the same brief with a fact sheet of names, numbers and one first-hand observation.
- The fact sheets must hold real, checkable material. The owner supplies it or approves the source. Do not invent facts for a sheet.
- Measures: blind reader preference between A and B, and counts of names and numbers per 100 words.
- Check each B sample against its sheet. A sample that adds a specific not on the sheet has fabricated it, and the rate of that is a result.

## 5. Reread

Purpose: test hypothesis 4.

- Condition A: one pass.
- Condition B: the same agent is asked to reread and revise its draft.
- Condition C: a second agent in a fresh context reads the draft without the brief or any style rules, reports where it stopped believing the text and what it wanted to know, and a revision follows from that report.
- Measures: blind reader preference across the three, and the profile.

## 6. Voice by description against voice by exemplar

Purpose: find the stronger lever for voice.

- Condition A: the voice specification from `BRIEF.md` as text.
- Condition B: human-written passages in the target voice, with no specification.
- Condition C: both.
- Needs human-written exemplars from the owner. Do not use model text as an exemplar.
- Measures: reader judgement of fit to the voice, reader preference on quality, and the profile against the exemplars' own profile.

## 7. Candidates and selection

Purpose: test whether choosing from several drafts beats one draft.

- Condition A: one draft.
- Condition B: several drafts from differently framed requests, with the pick made by a judge.
- The judge has to pass the calibration in `EVALUATION.md` first. Until then, human readers make the pick and the experiment measures the ceiling.
- The writing quality reward model from Chakrabarty, Laban and Wu has open weights and is a candidate ranker. Its training data is paragraph-level and leans towards creative writing, so check it against the readers before use.
- Measures: blind reader preference between A and B.

## 8. Voice swap

Purpose: test the brief's requirement that changing voice does not lower quality or add tells.

- Three voice specifications on the same briefs, one of them the test specification in `BRIEF.md`. The owner chooses or approves the other two.
- Measures: reader preference against baseline for each voice, and the profile for each voice.
- Reading: no voice should fall below baseline on preference, and no voice should produce a profile measure outside the human reference range. The test specification keeps several features rare, so look for an unusually flat distribution of sentence length.

## Human text on the same brief

- Russell et al. pair each model article with the human article that carried the same headline: github.com/jenna-russell/human_detectors
- Shaib et al. add span annotations to part of that set: github.com/cshaib/slop
- Check the licence and the source publications before copying any human text into the repository.
