# Evidence behind the procedure

Each step in `/prose:write` and `/prose:check` rests on a measurement from November 2026. The full write-ups, samples and scripts are in `prose-system/` in this repository: `RESULTS-EXP-1-3.md` and `RESULTS-RHYTHM.md`.

## What was measured

- 1,848 generated samples on 12 briefs in four kinds of prose, 10 samples per brief per condition. Most are from Claude Sonnet 5.5 running as a Claude Code subagent. Baselines were also taken on Opus 5.5, Fable 5.2 and Haiku 5.5.
- A reference of 133 human-written documents, 257,668 words, published before November 2022.
- All measures are surface features counted by a script. One reader judged six pairs in a pilot. No reader study has been run, so nothing here shows that a reader prefers the output.

## Findings the procedure uses

| Step | Finding |
| --- | --- |
| Plain prose, no headings or lists | The baseline used headings in 57% of samples and lists in 36%. Its mean sentence length was 14.6 words against 20.3 for the human texts. With the plain-prose line it was 19.0. |
| A full voice description | With the voice in `voices/plain.md`, sentence length was 18.8 words and commas 54.0 per 1,000 words, against 20.3 and 54.7 for the human texts. Words ending in -ly fell from 13.4 to 7.6 per 1,000. |
| No single rule about rhythm | "Vary your sentence length" raised the share of sentences of four words or fewer from 7.3% to 10.1%. "Do not use a short sentence to set up a longer one" raised the mean sentence length to 39.6 words. "Write mostly medium-length sentences" cut numbers per 100 words from 1.1 to 0.7. |
| No punctuation bans | Banning four joining marks cut digits per 100 words from 2.2 to 1.2. Banning one mark lowered the others as well. |
| No style revision pass | A second pass asked to fix rhythm raised the mean sentence length from 19.0 to 27.0 words and raised colons, semicolons and commas. |
| Several candidates, picked by measure | No instruction moved the alternation of short and long sentences. The lag-1 autocorrelation of sentence length was between -0.07 and -0.19 in every model condition and 0.04 in the human texts. Picking the best of five voice-file drafts by the ranking in `prose_profile.py` moved it from -0.14 to -0.05 and the short-then-long share from 9.1% to 7.0%, against a human 7.2%. This was computed on stored samples. |
| Material first, and a check of specifics | The voice file raised numbers per 100 words from 1.1 to 2.7. Whether those numbers were correct was not checked, so the procedure limits drafts to supplied material and lists any specific that is not in it. |
| No Haiku for drafting | 16 of 120 Haiku subagent runs called an unrelated tool by mistake, and two created empty cloud sessions. Sonnet did so in 7 of 480 runs, with Bash only. |

## Limits

- The ranking picks on the same measures it is judged by. It shows the pick is closer to the human profile and does not show the pick reads better.
- Dashes are close to absent in all four models, so this procedure does nothing about them. The human texts use 2.9 per 1,000 words.
- The models use about half as many names per 100 words as the human texts. Material supplied by the user is the only lever the procedure has on this.
- The reference texts were not written to the same briefs as the samples, and their human authorship rests on the publication date.
- The profile counts acronyms as names and identifiers such as "SHA-1" as numbers.
- The procedure was tested on English non-fiction of 300 to 2,000 words. It has not been tested on fiction, on very short text or on other languages.
- The voice in `voices/plain.md` is the only voice tested.
