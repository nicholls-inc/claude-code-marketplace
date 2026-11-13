# Results: experiments 1 to 3

Written 11 November 2026. This is the write-up the handoff asks for after experiments 1 to 3. It covers 1,368 generated samples and a human reference of 133 documents. No human reader has judged any of it. Every number here comes from `scripts/profile.py` and `scripts/analyse.py`, which measure surface features and do not measure quality.

## Summary

1. The four current Claude models write almost no dashes. The Sonnet 5.5 baseline has none in 120 samples. The human reference has 2.9 per 1,000 words. The em dash is not a tell of these models in this setting.
2. The brief's first failure claim, that a banned mark returns as a substitute, is not supported for an instruction given at generation time. Banning colons, parentheses or all four joining marks removed them, and the other marks and commas fell as well.
3. The brief's second failure claim is supported. "Vary your sentence length" shortened sentences and raised the share of very short sentences from 7.3% to 10.1%, and the samples show short sentences followed by long ones that pay them off.
4. The largest difference from human text is one no instruction in these experiments addressed. In every model and every condition, neighbouring sentences alternate in length. The lag-1 autocorrelation of sentence length is between -0.07 and -0.13 for the models and 0.04 for the human reference.
5. Bans on punctuation reduced specifics. With all four marks banned, digits per 100 words fell from 2.2 to 1.2. With headings and lists banned, they fell to 1.1.
6. Much of the baseline's choppy profile is formatting. A plain-prose instruction moved mean sentence length from 14.6 to 19.0 words, close to the human 20.3.

## How the samples were made

The handoff specifies one API call per sample that holds the condition's prompt and nothing else. No API key was available, and the owner chose Claude Code subagents run through workflows. That changes what the baseline is.

- Each subagent received the condition's prompt as its whole task. It also carried the Claude Code agent system prompt, tool definitions and the repository's `CLAUDE.md`. The system prompt asks for a concise report. It contains no instruction about punctuation or style.
- The baseline is therefore "a Claude Code subagent given a plain brief", not the bare model. The owner works in agent harnesses, so this baseline is relevant, and it is not the one the handoff describes.
- Sampling settings were the harness defaults and were not set by the experimenter.
- Each sample was copied from the subagent's transcript on disk by `harness/import_subagent.py`. No sample text passed through the orchestrating agent.
- All samples in the main analysis share one system prompt per model and saw no relayed user message. 120 earlier explanation samples saw the relayed word "Continue". They are stored separately and are compared below.

| Batch | Model served | Samples |
| --- | --- | --- |
| Experiments 1 to 3: `baseline`, `no_em_dash`, `no_joiners`, `vary_length` | claude-sonnet-5-5 | 480 |
| Follow-up: `no_colon`, `no_paren`, `plain_prose` | claude-sonnet-5-5 | 360 |
| Baseline on other models | claude-opus-5-5, claude-fable-5-2, claude-haiku-5-5 | 120 each |
| Kept aside: pilot, first workflow run, relayed "Continue" | claude-sonnet-5-5 | 168 |

Every cell has 10 samples per brief on the 12 development briefs. The held-out briefs were not used. Total usage was 107.2 million tokens: 88.3 million cache reads, 15.8 million cache writes and 3.1 million output.

### Human reference

`reference/human_reference.json` holds the same measures on 133 human-written documents, 257,668 words, published before November 2022, with at least 32 documents in each of the four kinds. Sources include The Conversation, MIT News, Paul Graham, Joel Spolsky, Pro Git, High Performance Browser Networking, Damn Interesting, NASA history pages and Wikipedia. The repository stores figures and URLs, not text.

The reference has three limits. The texts are on similar topics and were not written to the same briefs. Human authorship rests on the publication date. Extraction removed headings and lists, so the human text is running prose, and the fair comparison is with the `plain_prose` condition.

## Experiment 1: baseline

Expected: em dashes above 2.2 per 1,000 words in at least three kinds of prose. Result: wrong. The Sonnet baseline contains no em dash and no en dash.

| Measure | Human | Sonnet baseline | Sonnet plain prose | Opus | Fable | Haiku |
| --- | --- | --- | --- | --- | --- | --- |
| Mean sentence length, words | 20.3 | 14.6 | 19.0 | 14.5 | 15.6 | 14.9 |
| Sentences of 4 words or fewer | 3.4% | 7.3% | 2.4% | 8.1% | 6.8% | 8.0% |
| Short-then-long share | 7.2% | 14.3% | 10.4% | 14.3% | 14.2% | 15.0% |
| Lag-1 autocorrelation | 0.04 | -0.08 | -0.13 | -0.07 | -0.09 | -0.10 |
| Dashes per 1,000 words | 2.9 | 0.0 | 0.0 | 0.1 | 0.0 | 0.1 |
| Semicolons per 1,000 | 1.3 | 0.4 | 0.4 | 0.3 | 1.2 | 0.4 |
| Colons per 1,000 | 3.7 | 5.5 | 2.3 | 5.4 | 6.1 | 4.2 |
| Parentheses per 1,000 | 4.8 | 3.7 | 0.9 | 3.2 | 2.1 | 4.0 |
| Commas per 1,000 | 54.7 | 63.6 | 67.2 | 56.9 | 58.9 | 60.2 |
| Numbers per 100 words | 1.4 | 2.2 | 1.1 | 2.2 | 1.9 | 2.2 |
| Names per 100 words | 5.6 | 3.1 | 2.3 | 3.8 | 3.1 | 3.6 |
| Samples with headings | n/a | 57% | 0% | 40% | 28% | 28% |
| Samples with lists | n/a | 36% | 0% | 36% | 42% | 27% |

Raw dash characters, counted directly in the text: Sonnet 0 in 109,000 words, Haiku 0, Fable 2, Opus 25 in 114,000 words.

What the table shows:

- **The four models are close to each other.** On sentence length, short-then-long share and autocorrelation the models differ from each other by less than they differ from the human reference.
- **Dashes.** The human writers use them and the models do not. Half the human documents have more than 2.5 per 1,000 words.
- **Sentence length.** The baseline's 14.6 words is partly formatting. Headings and list items are short lines, and some are counted as sentences. In plain prose Sonnet averages 19.0 words.
- **Rhythm.** The autocorrelation is negative for every model and stays negative in plain prose. A negative value means a short sentence tends to sit next to a long one. The human value is 0.04, and 10% of human documents fall below -0.16. In plain prose the short-then-long share is 10.4% against the human 7.2%.
- **Names.** The models use about half as many capitalised names per 100 words as the human texts. The counter treats acronyms as names, which inflates technical text for both.
- **By kind, plain prose against human.** The short-then-long share is 12.3% against 10.3% in argument, 11.1% against 6.9% in explanation, 11.3% against 5.4% in narrative and 5.9% against 6.1% in technical description.

Examples of the short-then-long pattern in the `plain_prose` condition, which had no instruction on rhythm:

> That is the whole thing. A commit does not store a diff, and the changes you see in git show are computed on demand...

> Start with the track record. Decades of project data, from the Standish reports to the experience of anyone who has sat through a sprint retrospective...

> Reentry brought a final anxious silence. Normally, a few minutes of radio blackout occurs as the heat of reentry ionizes the air around the capsule.

## Experiment 2: banning marks

Expected: an em dash ban removes dashes and other marks make up part of the gap. Result: the em dash ban could not be tested, because the baseline has no dashes. The follow-up bans on marks the baseline does use showed no substitution.

Sonnet 5.5, per 1,000 words, 120 samples per condition, with 95% intervals:

| Condition | Dashes | Semicolons | Colons | Parentheses | Commas |
| --- | --- | --- | --- | --- | --- |
| `baseline` | 0.0 | 0.4 [0.3, 0.6] | 5.5 [5.1, 5.9] | 3.7 [3.4, 4.0] | 63.6 [62.2, 64.9] |
| `no_em_dash` | 0.0 | 0.3 [0.1, 0.4] | 4.6 [4.2, 5.0] | 3.0 [2.7, 3.4] | 63.3 [62.3, 64.3] |
| `no_colon` | 0.0 | 0.0 [0.0, 0.0] | 0.3 [0.2, 0.5] | 0.9 [0.7, 1.2] | 60.1 [58.9, 61.3] |
| `no_paren` | 0.0 | 0.1 [0.0, 0.2] | 1.3 [1.1, 1.6] | 0.0 [0.0, 0.0] | 58.5 [57.1, 59.7] |
| `no_joiners` | 0.0 | 0.0 [0.0, 0.0] | 0.1 [0.0, 0.1] | 0.0 [0.0, 0.1] | 52.4 [51.4, 53.4] |

- **A ban on one mark lowered the others.** Banning colons cut parentheses from 3.7 to 0.9. Banning parentheses cut colons from 5.5 to 1.3. Commas fell in every ban condition.
- **A ban on an absent mark still changed the text.** `no_em_dash` had no dashes to remove. It cut the share of samples with headings from 57% to 32% and with lists from 36% to 20%.
- **Part of the fall is formatting.** Headings and lists carry colons and parentheses. Under `no_colon` 7% of samples had lists.
- **Sentence fragments and comma splices did not rise.** Under `no_joiners` the share of sentences of four words or fewer was 6.7% against 7.3%, commas fell, and mean sentence length stayed at 14.6 words.
- **What replaced the marks was wording.** "called" rose from 0.59 to 0.98 per 1,000 words under `no_joiners`, where a parenthetical such as "(COP)" became "called the coefficient of performance".
- **Specifics fell.** Under `no_joiners` digits per 100 words fell from 2.2 to 1.2, with no rise in spelled-out numbers, and names fell from 3.1 to 2.2.

Expected results for the follow-up, checked: colons and parentheses fell as predicted, and the other marks did not rise as predicted. The prediction that commas would rise under `no_paren` was wrong.

This does not contradict what the owner saw. The brief describes tools that rewrite an existing draft. In a rewrite the banned mark has a position, and a substitute can take it. These experiments gave the instruction before any draft existed. A rewrite test is listed under next steps.

## Experiment 3: "vary your sentence length"

Expected: the instruction raises the coefficient of variation and the short-then-long share and lowers the autocorrelation. Result: the first two held and the third did not move beyond its interval.

| Measure | `baseline` | `vary_length` |
| --- | --- | --- |
| Mean sentence length, words | 14.6 [14.4, 14.8] | 12.9 [12.7, 13.0] |
| Coefficient of variation | 0.55 [0.55, 0.56] | 0.58 [0.57, 0.59] |
| Sentences of 4 words or fewer | 7.3% [6.8, 7.8] | 10.1% [9.5, 10.7] |
| Short-then-long share | 14.3% [13.7, 14.9] | 16.1% [15.5, 16.7] |
| Lag-1 autocorrelation | -0.08 [-0.10, -0.06] | -0.11 [-0.13, -0.09] |

The model met the instruction by adding short sentences. The most frequent sentences of three words or fewer in `vary_length` include "Why?", "It sounds impossible.", "Fair.", "It is not." and "They pad." Examples from the samples:

> They pad. They cut corners to hit the number.

> The tank split open. A wave of molasses, perhaps fifteen feet high at its crest, burst outward at roughly thirty-five miles an hour.

> It did not work. In one of the earliest large class action suits in Massachusetts, residents sued...

The effect was largest in narrative, where the short-then-long share rose from 13.1% to 17.2% against a human 5.4%.

## The relayed message

120 explanation samples were written by subagents that saw the word "Continue" before the prompt. Compared with the rerun without it, the baseline averaged 790 words per sample against 833, and the intervals do not overlap. Mean sentence length, marks, numbers and the short-then-long share overlap. View: one neutral word in the context moved length by about 5%, or two runs an hour apart differ by that much. The data cannot tell these apart.

## What this means for the brief and the findings

These are views.

- **The brief's list of tells is partly out of date for these models.** The em dash is absent. Headings, lists and the alternation of short and long sentences are present.
- **Failure claim 1** is not supported at generation time on these models. It remains untested for rewrite passes.
- **Failure claim 2** is supported, and the pattern it describes is already in the baseline. The instruction adds to it.
- **Failure claim 3**, that a draft free of listed patterns still reads as machine-written, cannot be tested without readers. The autocorrelation result gives it a measurable candidate: a feature no pattern list names that separates every model condition from the human reference.
- **Surface instructions have content effects.** Bans lowered the count of numbers and names. A system that edits surface form needs a check that specifics survive, which supports the material stage in the design direction.
- **The profile is useful as an alarm and has known faults.** It counts acronyms as names and identifiers such as "SHA-1" as numbers, and it reads an unmarked heading line as a sentence.

## Limits

- No human reader has seen any sample. Nothing here shows which condition reads better or which is identified as model text.
- The baseline carries the Claude Code agent context. A bare API baseline may differ, in formatting above all.
- Twelve briefs, one prompt wording per condition. A differently worded ban may behave differently.
- The human reference is not matched to the briefs, and its texts had headings and lists removed.
- Sampling settings were not controlled.
- 7 of 480 Sonnet runs in experiments 1 to 3 and 16 of 120 Haiku runs used a tool. Two Haiku runs created empty cloud sessions on the owner's account. See `LOG.md`.

## Next steps, for the owner to choose

1. **Reader protocol.** Build the blind reader page in `EVALUATION.md`. This needs the answer to open question 3 on readers.
2. **Rewrite test for claim 1.** Give a model a draft that contains dashes and ask it to remove them, then measure the marks at the same positions. This needs a source of drafts with dashes, since the models do not produce them.
3. **Rhythm.** Test whether any instruction or exemplar moves the autocorrelation towards the human value without adding short hooks. This is the first candidate for the system's voice stage.
4. **Bare API baseline.** Rerun the baseline through the API when a key is available, to measure how much of the profile comes from the agent context.
5. **Experiments 4 to 8.** These need the owner's answers to open questions 3 to 6.
