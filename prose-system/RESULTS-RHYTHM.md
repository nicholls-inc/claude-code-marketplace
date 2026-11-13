# Results: rhythm test

Written 11 November 2026. It follows `RESULTS-EXP-1-3.md` and uses the same method: Sonnet 5.5 subagents, 12 development briefs, 10 samples per brief per condition, measured with `scripts/analyse.py`. No human reader has judged any sample.

## Summary

1. No instruction moved the alternation of short and long sentences towards the human value. In four conditions it moved further away.
2. Each instruction about rhythm fixed the feature it named and produced a new outlier. "Do not use a short sentence to set up a longer one" removed short sentences and raised the mean sentence length to 39.6 words. A revision pass with the same aim raised it to 27.0.
3. The revision pass joined sentences with colons, semicolons and commas, and all three rose. This is the substitution the brief describes, and it appeared in a rewrite where it had not appeared at generation time.
4. The test voice specification from the brief gave the profile closest to the human reference on most measures. It doubled the count of numbers and halved the count of words ending in -ly. The alternation did not change.
5. The alternation result holds when document length is controlled for.

## Conditions

All conditions include "Write it as plain prose with no headings or lists", so they compare with the stored `plain_prose` samples and with the human reference, which is running prose.

| Condition | Added to the plain-prose prompt |
| --- | --- |
| `plain_prose` | nothing |
| `rhythm_content` | "Let the length of each sentence follow what it has to say. Do not use a short sentence to set up a longer one." |
| `rhythm_medium` | "Write mostly medium-length, complete sentences." |
| `voice_spec` | the test voice specification from `BRIEF.md`, word for word |
| `revise_rhythm` | a second subagent reads one stored `plain_prose` sample and revises it: "Revise it so that the length of each sentence follows what it has to say. Where a short sentence sets up a longer one, combine them or recast them. Keep the content, the facts and the order." |

## Results

120 samples per condition. The human column is the reference of 133 documents.

| Measure | Human | `plain_prose` | `rhythm_content` | `rhythm_medium` | `voice_spec` | `revise_rhythm` |
| --- | --- | --- | --- | --- | --- | --- |
| Mean sentence length, words | 20.3 | 19.0 | 39.6 | 22.0 | 18.8 | 27.0 |
| Coefficient of variation | 0.55 | 0.50 | 0.45 | 0.43 | 0.47 | 0.47 |
| Sentences of 4 words or fewer | 3.4% | 2.4% | 0.2% | 0.7% | 1.7% | 0.7% |
| Short-then-long share | 7.2% | 10.4% | 2.7% | 6.5% | 9.1% | 4.5% |
| Lag-1 autocorrelation | 0.04 | -0.13 | -0.15 | -0.19 | -0.14 | -0.16 |
| Semicolons per 1,000 | 1.3 | 0.4 | 0.6 | 0.2 | 0.0 | 1.1 |
| Colons per 1,000 | 3.7 | 2.3 | 1.1 | 0.8 | 0.6 | 4.1 |
| Parentheses per 1,000 | 4.8 | 0.9 | 0.7 | 0.1 | 0.1 | 0.9 |
| Commas per 1,000 | 54.7 | 67.2 | 65.5 | 62.8 | 54.0 | 76.1 |
| Numbers per 100 words | 1.4 | 1.1 | 1.2 | 0.7 | 2.7 | 1.1 |
| Names per 100 words | 5.6 | 2.3 | 2.5 | 2.0 | 3.3 | 2.4 |

Intervals are in `results/rhythm-sonnet.md` and `results/rhythm-revise.md`.

### Against the expected results

- `rhythm_content`. Expected: very short sentences below 2%, short-then-long at 8% or less, autocorrelation below zero. All three held. Not expected: a mean sentence length of 39.6 words, which is above the 90th percentile of the human documents at 26.8.
- `rhythm_medium`. Expected: coefficient of variation below 0.45 and autocorrelation below zero. Both held. Numbers per 100 words also fell from 1.1 to 0.7.
- `voice_spec`. Expected: mean above 20 words, coefficient of variation below 0.45, parentheses below 0.5, autocorrelation below zero. The first two were wrong, at 18.8 and 0.47. The last two held.
- `revise_rhythm`. Expected: short-then-long falls, autocorrelation moves towards zero, digits hold within 10%. The first and third held. The autocorrelation moved from -0.13 to -0.16.

### The voice specification

- Sentence length 18.8 words and commas 54.0 per 1,000 are close to the human 20.3 and 54.7.
- Numbers per 100 words rose from 1.1 to 2.7 and names from 2.3 to 3.3. The specification asks for numbers, names and concrete facts. Whether the added numbers are correct has not been checked.
- Words ending in -ly fell from 13.4 to 7.6 per 1,000 words. The specification asks for adverbs to be rare.
- Parentheses, colons and semicolons fell close to zero, below the human rates. The specification asks for parentheticals to be rare and says nothing about colons or semicolons.
- Spelling was already British in the `plain_prose` samples, at 5.3 British forms per 1,000 words against 0.4 American, so the instruction had little to change.
- The short-then-long share fell from 10.4% to 9.1%, against 7.2% for the human texts. The specification names this pattern and the model reduced it by about an eighth.

### The alternation, with length controlled

The lag-1 autocorrelation is biased downwards in short documents, and the samples are shorter than the human documents. Two checks were run. Human documents were cut into pieces of 20 and 45 sentences. Each set was also compared with the same sentences in shuffled order, which gives the value expected when order carries no information.

| Set | Sentences per document | Lag-1 | Shuffled, mean and 95% range | Lag-1 minus shuffled |
| --- | --- | --- | --- | --- |
| Human, full documents | 96.2 | 0.04 | -0.01 [-0.03, 0.01] | 0.05 |
| Human, cut to 45 sentences | 45.0 | 0.02 | -0.02 [-0.04, 0.00] | 0.04 |
| Human, cut to 20 sentences | 20.0 | -0.03 | -0.05 [-0.07, -0.03] | 0.02 |
| Sonnet `baseline` | 60.6 | -0.08 | -0.02 [-0.04, 0.01] | -0.06 |
| Sonnet `plain_prose` | 45.8 | -0.13 | -0.02 [-0.05, 0.00] | -0.11 |
| `voice_spec` | 45.9 | -0.14 | -0.02 [-0.05, 0.00] | -0.11 |
| `rhythm_medium` | 39.0 | -0.19 | -0.03 [-0.05, 0.00] | -0.16 |
| `rhythm_content` | 23.1 | -0.16 | -0.04 [-0.07, -0.01] | -0.11 |
| `revise_rhythm` | 32.3 | -0.16 | -0.03 [-0.06, 0.00] | -0.13 |
| Opus `baseline` | 64.9 | -0.07 | -0.02 [-0.04, 0.01] | -0.06 |
| Fable `baseline` | 54.6 | -0.10 | -0.02 [-0.04, 0.00] | -0.08 |

Human text sits above its shuffled value at every length: neighbouring sentences are a little more alike than chance. Every model set sits below its shuffled range: neighbouring sentences are less alike than chance. The human rows use the 142 extracted texts in the cache, which include 9 that the reference later dropped.

## What this means for the design

These are views.

- **Rhythm cannot be fixed by naming it.** Each instruction traded one outlier for another, and a rewrite pass did the same and added joining marks. This supports the finding in the first write-up that a positive style rule produces a new tell, and extends it to revision.
- **A full voice description did better than a single rule.** The specification names several features at once, and no single measure ran to an extreme. Sentence length and commas landed near the human values.
- **The alternation is the open problem.** It is present in every model and condition and no prompt-level lever tested here moved it. Two levers remain untested: human exemplars, which need the owner's material, and selection among several candidates by a measured value.
- **The measures are not quality.** A sample can match the human profile and read badly. Readers decide whether any of these differences matter.

## Limits

- One wording per condition, on one model.
- The revision pass read its draft from a file, so each reviser used the Read tool. The other conditions used no tool in 353 of 360 runs.
- The numbers that `voice_spec` added were not checked against sources.
- The human reference is not matched to the briefs.
