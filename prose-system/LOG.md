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

### Later the same day: route change and pilot

- The owner chose Sonnet subagents in Claude Code as the generator, in place of API calls, and approved both brief sets. They are frozen at version 1.
- This departs from the handoff, which asks for a call that holds the condition's prompt and nothing else. A subagent also carries the Claude Code agent system prompt, tool definitions and `CLAUDE.md`. One line in that system prompt asks for "a concise report". Every record from this route is marked `"route": "claude-code-subagent"` and stores a hash of the system prompt.
- `harness/import_subagent.py` copies each sample from the subagent's transcript on disk, so the text does not pass through the orchestrating agent.
- Pilot: brief `dev-exp-1`, four conditions, two samples each, 8 subagents, stored under `samples/subagent/pilot/`. These are pilot samples and are not the experiment 1 baseline.

Pilot results:

- Model served in all 8: `claude-sonnet-5-5`. The system prompt hash was the same in all 8.
- Tokens, summed over 8 runs: 587,337 input tokens written to the prompt cache, 72,116 read from it, 18 uncached input, 16,837 output including thinking. That is 676,308 in total and about 84,500 per run. About 72,100 of each run is the fixed subagent context.
- The 8 runs started together and none read the cache that another had written. The one cache read came from the second API message of a single run.
- Length: 623 to 742 words against a request for about 600.
- 7 of 8 returned the piece and nothing else. 1 of 8, `vary_length` sample 1, used the Bash tool to save a copy, put a word count in its title and added a line about the saved file. It is stored with `tools_used: ["Bash"]`.
- 5 of 8 used bold headings and 4 of 8 used lists. `profile.py` counted a bold heading line as a sentence. It now drops a line that is entirely bold.
- No sample in any condition, the baseline included, contains an em dash. This goes against the expected result for experiment 1 above. Two baseline samples on one brief are too few to settle it.


### First workflow run: 40 samples, and a context problem

- The owner approved the 480 runs, asked for them to be staggered through workflows, and ruled that tool use by a subagent is valid. Tool use is recorded per run and not rejected.
- Workflow run 1: brief `dev-exp-1`, four conditions, ten samples each, four subagents at a time. All 40 returned. Stored under `samples/subagent/workflow-run1/`.
- Problem found on import. The workflow harness relays the owner's launching message to every subagent, ahead of the condition's prompt. All 40 subagents were shown this text: "Go ahead with the 480 runs but batch or stagger them so we can spread usage out. Use a workflow or multiple workflows. What's wrong with using tools? Agents will do that IRL. Fine with leaving baseline prompt as is." The condition's prompt then arrives inside a frame that calls it a task computed by a workflow script.
- So these 40 samples were written by subagents that knew they were one of 480 runs and that a baseline prompt existed. They were not shown the brief, the findings or any list of tells. No sample text contains "480", "baseline", "experiment", "workflow" or "sample".
- The system prompt hash is the same in all 40 and differs from the pilot's, so a workflow subagent and an Agent-tool subagent do not share a system prompt. The pilot and this run are not comparable with each other.
- A workflow launched after a later owner message would relay that later message. Runs launched under different messages would differ in context.
- Tokens over 40 runs: 259,264 written to cache, 2,646,866 read from cache, 80 uncached input, 75,486 output including thinking. About 74,500 per run. 39 of 40 runs read the cache, so staggering in waves of four did share it.
- Model served in all 40: `claude-sonnet-5-5`. No run used a tool. Length 547 to 782 words.
- Pooled profile for this one brief, about 6,500 to 7,100 words per condition. Em dashes are 0.0 per 1,000 words in every condition, the baseline included. Combined joiners per 1,000 words: baseline 7.3, `no_em_dash` 2.0, `no_joiners` 0.0, `vary_length` 2.5. One brief is not a result for experiments 1 to 3.
- The other 440 runs are not started. The owner decides how to handle the relayed message first.

### The 480 runs: relay differs between workflows

- The owner chose to rerun all 480 under one neutral message and replied "Continue".
- Group 1, the three explanation briefs, was launched in the turn that answered that message. Its 120 subagents were shown "Continue" as a relayed user request, then the condition's prompt.
- Groups 2 and 3, the argument and narrative briefs, were launched in turns that followed a workflow's completion notice. Their subagents were shown no relayed request, only the condition's prompt inside the workflow frame. The harness relays a user message only when the launching turn answers one.
- The no-relay context is the one closer to the handoff. Group 1 is therefore being rerun without the relay after group 4, so that all 480 runs share one context. This adds 120 runs to the 480 the owner approved.
- The 120 "Continue" samples are kept under `samples/subagent/exp1-3-relayed-continue/`. Comparing them with the rerun shows whether the relayed word changes the profile.

### 480 runs complete, and follow-up runs

- All 480 samples for experiments 1 to 3 are stored under `samples/subagent/exp1-3/`: 12 briefs, 4 conditions, 10 samples each. All were served by `claude-sonnet-5-5` with one system prompt and no relayed user message. 7 runs used Bash. None touched the experiment's files. Tokens: 37.5 million, of which 34.7 million were cache reads and 1.0 million were output.
- Tables are in `results/exp1-3-sonnet.md`, produced by `scripts/analyse.py`.
- The owner said usage is not a constraint and that further runs or tests are welcome where they add value.

Follow-up runs, with expected results stated before running. These are the agent's views.

- Baseline on three more models through the same route: Opus, Fable and Haiku subagents, 120 samples each. Expected: no more than one of the three exceeds 2.2 dashes per 1,000 words. Sentence length means sit between 12 and 17 words for all three, below the Brown mean of 20.0.
- `no_colon` on Sonnet. The baseline uses 5.5 colons per 1,000 words. Expected: colons fall below 0.5. Semicolons and parentheses do not rise beyond their baseline intervals. The share of samples with lists falls. If the first failure claim in the brief held for this model, another joining mark would rise by a large part of the 5.5.
- `no_paren` on Sonnet. The baseline uses 3.7 per 1,000 words. Expected: parentheses fall below 0.3 and commas per 1,000 words rise, because a parenthetical recast in running prose is usually set off by commas.
- `plain_prose` on Sonnet. Expected: headings and lists fall to near zero. Colons and parentheses fall by more than half in the technical briefs. Numbers and names per 100 words fall by less than a fifth.

### Haiku baseline, and tool side effects

- Haiku baseline: 120 samples under `samples/subagent/models/haiku/`, all served by `claude-haiku-5-5`, no relayed message. Its system prompt hash differs from Sonnet's.
- 16 of the 120 Haiku runs called a tool other than the handback, against 7 of 480 for Sonnet. The Haiku calls were misfires while looking for the handback tool, mostly with empty arguments. None wrote to the repository or read the experiment's files.
- Two of those misfires had effects outside the run. Each called `create_session` with no arguments and created an empty Claude Code cloud session on the owner's account: `session_01NFQZoWACkoQUQP95Sxt2mY` and `session_01NFQZoTogBRsaiHxi48iUYr`. Both were idle with 0 tokens used. The orchestrating agent tried to archive both and the session's permission check refused the action, so both are still there for the owner to archive.
- One run queued a suggested-task card with placeholder text and withdrew it itself. Two runs listed the account's environments, one listed its repositories and one listed its sessions. Those results went to the subagent only.
- The owner's ruling that tool use is valid stands for the samples. The side effects are a separate matter: a subagent given only a writing brief can still reach every tool the session has.

### Rhythm test, expected results stated before running

The owner read the write-up of experiments 1 to 3 and asked for the rhythm test and the reader page. All conditions are on Sonnet subagents, 12 development briefs, 10 samples each, and are compared with the stored `plain_prose` samples and the human reference. These are the agent's views.

- `rhythm_content`: plain prose, plus "Let the length of each sentence follow what it has to say. Do not use a short sentence to set up a longer one." Expected: sentences of four words or fewer fall below 2% and the short-then-long share falls to 8% or less. The lag-1 autocorrelation stays below zero.
- `rhythm_medium`: plain prose, plus "Write mostly medium-length, complete sentences." Expected: the coefficient of variation falls below 0.45 against a human 0.55, which is a new outlier. The autocorrelation stays below zero.
- `voice_spec`: plain prose, plus the test voice specification from the brief. Expected: mean sentence length above 20 words, coefficient of variation below 0.45, parentheses below 0.5 per 1,000 words, autocorrelation below zero.
- `revise_rhythm`: a second subagent revises each of the 120 stored `plain_prose` samples for rhythm. Expected: the short-then-long share falls against its drafts, the autocorrelation moves towards zero by less than half the gap to the human value, and digits per 100 words stay within 10% of the drafts.

### Reader page

- `reader/page.html` is the blind paired-comparison page from `EVALUATION.md`, published as a private artifact at https://claude.ai/artifact/KtGe2FonWVA5uuxtV9YQ6b. It shows one pair at a time, asks which passage the reader prefers, who wrote each passage and one line of reason, and records the reader's use of language models once.
- Answers are stored in the artifact's database, one document per reader under `responses/`. A reader can read and write only their own document. The owner and editors can read all of them.
- `reader/build_round.py` builds a round. `rounds/<round>/trials.json` is published and holds opaque ids, the brief and two passages. `rounds/<round>/key.json` stays in the repository and says which source is A and which is B.
- Round `r0` is a pilot for trying the page: six pairs of `baseline` against `plain_prose` on the six shortest briefs, 7,171 words. Both passages in every pair are model text. It is not one of the pairings in `EVALUATION.md`.
- Not tested: a reader other than the owner taking part. The page has not been opened by a person yet.
