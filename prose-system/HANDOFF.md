# Handoff: AI prose system

You are taking over a project from an investigation session that had web access and no API access. That session surveyed the research and the existing tools, measured one human reference corpus, and proposed an evaluation method and a design. It generated no model text and ran no reader tests. Your job is to run the experiments, set up the evaluation, and then build the system the results support.

## Read in this order

1. `BRIEF.md`. The owner's original brief, unchanged. It is the authority. Where any other file here conflicts with it, the brief wins.
2. `FINDINGS.md`. What the investigation found, and four points where it disagrees with the brief.
3. `EXPERIMENTS.md`. Eight experiments, none run.
4. `EVALUATION.md`. The proposed reader protocol, judge calibration and brief sets.
5. `OPEN_QUESTIONS.md`. Questions only the owner can answer.

Also in the package:

- `scripts/profile.py`. Measures surface features of raw text samples. Standard library only.
- `scripts/brown_reference.py` and `reference/brown_reference.json`. The human reference measurement, reproducible.

## Status of each step in the brief

| Step in the brief | Status |
| --- | --- |
| 1. Investigate before designing | Done from published sources. See the caveats below. |
| 2. Run experiments | Not started. One measurement on human text exists. |
| 3. Set up an evaluation method | Proposed on paper. No instrument built, no readers recruited, no judge calibrated. |
| 4. Keep a baseline | Not started. |
| Build the system | Not started. A design direction exists and is untested. |

## What to do

1. Confirm the environment: an API key, the model you will use as generator, and its exact version string. Record them.
2. Put the questions in `OPEN_QUESTIONS.md` to the owner. Several block later steps. Do not guess the answers.
3. Build the generation harness. Each sample is one API call with a fresh context. Store every sample with its prompt, model, settings, condition, brief and timestamp.
4. Draft the 12 development briefs and the 24 held-out briefs. Stop for the owner's approval, then freeze and version them.
5. Run experiments 1 to 3 on the development briefs. They need no human readers. Write up the results and stop for the owner to read them.
6. Build the blind reader instrument in `EVALUATION.md` and run experiments 4 to 8 as readers become available.
7. Calibrate a model judge against the reader data, if one is wanted.
8. Write up the results. State which parts of the design direction in `FINDINGS.md` the results support and which they do not. Stop for the owner before building the system.
9. Build the system in the form the results support, with the feedback ledgers and the held-out check described in the brief.

## Rules

- Every claim about what works has to rest on outputs you produced and examined. This is from the brief.
- Do not write test samples yourself. You have read the brief, the list of tells and the hypotheses, so your own prose is neither a baseline nor a fair test. Samples come from fresh-context calls that contain only the condition's prompt.
- The design direction in `FINDINGS.md` is a view. Treat each stage as a hypothesis. If the results go against it, say so and follow the results.
- Human readers are the measure. The profile script is an alarm and not a score. A model judge is used only after calibration against readers.
- Do not use AI-text detectors as the measure. Ask the owner before using one for any purpose.
- The held-out briefs are never used to write or adjust a prompt.
- Keep feedback about quality separate from feedback about voice.
- Do not invent facts, names, quotations or sources in fact sheets, exemplars or reports. Where material is missing, ask the owner.
- Report null results and results that go against the brief or the findings.
- State your assumptions as you go.

## Stop points

Stop and wait for the owner at each of these.

- After the brief sets are drafted, before freezing them.
- After experiments 1 to 3.
- Before spending on anything large, such as a long batch of generations or paid readers.
- Before building the system.

## Caveats on the findings

Check these before relying on them.

- The figures from Reinhart et al., Russell et al., Zhang et al., both Chakrabarty papers and Paech et al. were taken from abstracts and from the text shown in search results. Only Shaib et al. was read in full. Read the papers before building on a specific number.
- The statement that the verbalised sampling paper does not measure reader preference for non-fiction was not checked against the full paper.
- The humanizer pattern counts from 24 to 35 come from its README and change log as shown in search results. The current file on the main branch numbers 26.
- The practitioner report on substitution is one person's account and has no data attached.
- The cited studies used GPT-4o, Claude 3.5 Sonnet, o1 and Llama 3. Experiment 1 is the check on whether current models behave the same way.
- The Brown reference is from 1961 and uses the corpus's own tokenisation. `profile.py` splits raw text itself, so the two are close and not identical. Build a current reference with `profile.py` before using the profile as an alarm.
- `profile.py` has been smoke-tested on three markdown files and nothing else. Its sentence splitter is simple. Check it on real samples and fix it where it miscounts.

## Prose in your own reports

The owner reads the write-ups. Use British spelling, plain words and complete sentences of moderate length. Lead with numbers and names. Label any view as a view. Keep em dashes, parentheticals and adverbs rare.
