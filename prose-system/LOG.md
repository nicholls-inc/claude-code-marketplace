# Run log

Kept by the agent that took over from the investigation session. Newest entry last.

## 11 November 2026

### Environment

- Session: Claude Code in a cloud container, repository `nicholls-inc/claude-code-marketplace`, branch `claude/practical-einstein-hr7cam`.
- API access: none. `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` are not set, and the `ant` CLI is not installed. No sample has been generated.
- Generator model and version: not chosen. This is open question 1.
- Python 3.13.18. `anthropic` SDK 2.8.0 installed with pip for the harness. The profile scripts use the standard library only.

### Done

- Package copied into `prose-system/` unchanged, apart from `scripts/profile.py`.
- `harness/generate.py` written. One Messages API call per sample, holding the condition's prompt and nothing else. `harness/test_generate.py` runs it against a stub client and passes 14 checks. It has not made a real API call.
- `conditions/exp1-3.json` written: `baseline`, `no_em_dash`, `no_joiners`, `vary_length`.
- `briefs/dev.json` with 12 briefs and `briefs/held-out.json` with 24 briefs, both marked `draft-1` and not frozen. The harness refuses an unfrozen set.
- `scripts/profile.py` checked on 22 constructed cases in `scripts/test_profile.py`. 7 failed and 6 were faults in the script. The fixes:
  - Hard-wrapped text was split into a sentence at every line break. Lines inside a paragraph are now joined.
  - `J. K. Rowling` and `U.S. Army` were split into sentences at each full stop. Single capital initials and upper-case dotted abbreviations no longer end a sentence.
  - `3.5` counted as two words and two numbers. It now counts as one.
  - `a.m.`, `e.g.` and `U.S.` counted as two words each. Each now counts as one.
  - `I'm`, `I've` and `I'd` counted as names. They no longer do.
  - The seventh failure was a wrong expected value in the test.
- Licence check on the two sources of human text on the same brief:
  - `jenna-russell/human_detectors` carries an MIT licence on the repository. The human articles come from news publishers, and the one example in the README names the Associated Press. The README says nothing on the copyright of those articles. No article has been copied into this repository.
  - `cshaib/slop` holds a licence file and a README that reads "Coming soon!". It has no data at the time of checking.

### Not done

- The check of `profile.py` on real samples. It needs generated samples and current human text.
- A current human reference built with `profile.py`.
- The reading of the cited papers in full. The figures in `FINDINGS.md` still carry the caveats in `HANDOFF.md`.

### Assumptions

- The brief sets follow the proposal in open question 2: four kinds of prose, 300 to 2,000 words. The owner has not confirmed it.
- The narrative briefs are accounts of documented public events. A personal piece needs the owner's own material, which the baseline prompt cannot hold without it, so no brief asks for one.
- The baseline prompt gives the topic, the length and the reader. It says nothing about headings or lists. `profile.py` drops headings and counts each list item as a sentence.
- Sampling settings are the API defaults. The harness sends no temperature, thinking or effort setting. The default effort differs between models, and the record stores the model identifier so the default can be looked up.
- No refusal fallback is set. A fallback would have a second model write the sample. A refused call is stored and marked not valid.
- Experiments 1 to 3 at 10 samples per brief come to 480 calls: 12 briefs, 4 conditions, 10 samples. The briefs ask for 9,700 words in total, so one pass over the 12 briefs is about 9,700 words and the 480 calls are about 388,000 words of output before thinking tokens.

### Expected results for experiments 1 to 3, stated before running

These are the agent's views. None rests on generated output.

- Experiment 1. The baseline uses em dashes above the Brown median of 2.2 per 1,000 words in at least three of the four kinds of prose. Names and numbers per 100 words are lowest in the argument briefs.
- Experiment 2. `no_em_dash` brings em dashes close to zero, and the combined rate of the four marks falls by less than the em dash rate fell, with colons and semicolons making up part of the gap. `no_joiners` brings all four marks close to zero and raises the count of sentence fragments and comma splices against the baseline.
- Experiment 3. `vary_length` raises the coefficient of variation of sentence length and raises the short-then-long share against the baseline. The lag-1 autocorrelation falls.
