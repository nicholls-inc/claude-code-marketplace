# AI prose system: investigation findings

Written 11 November 2026 in a chat session with no API access. This is a copy of the findings document the owner has read.

## Scope

This document covers step 1 of the brief and the write-up the brief asks for before anything large is built. It surveys existing tools and published research, checks the brief's hypotheses and failure claims against that evidence, and proposes an evaluation method. The system itself is not built yet.

Done in that session: a survey of six tools and seven papers, a reading of two rewrite skills from their source files, and one measurement on 357 human-written documents. Not done: generation of model text, comparison with human text on the same brief, and any reading by human readers. Section 5 gives the reason and lists the eight experiments that are queued.

The main finding is that the published evidence places the fault in what the draft contains and in the model's default distribution, and that the tools in use act on sentence patterns after the draft exists.

## 2. Causes

The published evidence supports hypotheses 1 and 3, weakens hypothesis 2 as the brief states it, and does not test hypothesis 4. It also points to a limit the brief does not list, which is covered under the table.

| Hypothesis in the brief | Status | Evidence |
| --- | --- | --- |
| 1. Preference tuning narrows output towards one register | Supported | Reinhart et al. compared Llama 3 base models with their instruction-tuned variants on Biber's feature set. The tuned models used present participial clauses at 2 to 5 times the human rate and nominalisations at 1.5 to 2 times, and the base models sat closer to human rates. Zhang et al. found that annotators in preference datasets favour more typical text and tied this to mode collapse after post-training. |
| 2. Next-token prediction favours the stock phrase | Weakened | Base models predict the next token too, and in Reinhart et al. they matched human feature rates more closely than tuned models. Zhang et al. report that aligned models keep much of the base model's diversity and that a prompt asking for a distribution of responses raised diversity in creative writing by 1.6 to 2.1 times. The evidence places the stock phrase in the tuned distribution, not in prediction as such. |
| 3. No observation of its own, so space is filled with general claims | Supported as what readers penalise | In Shaib et al. three professional copy-editors marked spans in 213 news articles and 123 question-answering passages. Relevance, density and tone were the strongest predictors of a slop label. Russell et al. report that expert readers cite vocabulary first and also formality, originality and clarity. Neither paper shows that missing observation is the cause. |
| 4. Single pass with no reread | Not tested | The nearest evidence is against a naive second pass. In Russell et al. a majority vote of five expert readers misclassified 1 of 300 articles, and that set included articles the models had paraphrased or been prompted to humanise. |

The limit the brief does not list comes from Chakrabarty, Ginsburg and Dhillon. With in-context prompting, 28 MFA-trained readers disfavoured model text against expert human writers at an odds ratio of 0.16 for stylistic fidelity and 0.13 for quality. After fine-tuning on each author's works the same readers favoured model text at 8.16 and 1.87. The models were GPT-4o, Claude 3.5 Sonnet and Gemini 1.5 Pro, and the task was literary style emulation in excerpts of up to 450 words.

View: the tells belong to the default tuned distribution, and instructions alone moved output only part of the way out of it. The brief rules out fine-tuning, so the system has to find other ways to shift the distribution, and section 7 lists the candidates.

## 3. Existing tools

The tools examined fall into two groups: rewrite passes that work from a pattern catalogue, and research methods that change how text is sampled or selected. The first two in the table were read from their source files, and the rest from their papers or published prompts.

| Tool | What it does | Where it falls short |
| --- | --- | --- |
| stop-slop (Hardik Pandya), installed in the owner's plugins | Eight rules and three reference files of 1,634 words. It bans all adverbs, all passive constructions and all em dashes. It asks the model to score its draft from 1 to 10 on five dimensions and revise below 35 of 50. | The em dash rule names the replacement: commas or full stops. The rhythm rule says to mix sentence lengths and break any run of three matching sentences, which is the positive rule the brief describes. The score comes from the model that wrote the draft and has not been checked against readers. |
| blader/humanizer | 26 numbered patterns in six groups, taken from Wikipedia's Signs of AI writing. The model marks tells, rewrites, checks and rewrites again. Details must come from the writer, and a writing sample overrides the pattern rules. | The dash rule lists the substitutes as full stop, comma, colon or parentheses. The final step says to vary sentence length. The check step names five tells that most often survive a rewrite. The pattern count has moved between 24 and 35 across versions. |
| de-slopify (Dicklesworthstone) | A single review prompt that hunts for tells in finished text. | The prompt tells the model to replace an em dash with a semicolon or a comma. |
| Antislop sampler (Paech et al.) | Detects a banned string during generation, backtracks to its first token, lowers that token's probability and resamples. It suppressed more than 8,000 patterns, where plain token banning became unusable at 2,000. | It needs control of the sampler, which the hosted APIs in the brief do not give. Its quality scores come from a model-judged rubric. It is still a list. |
| Verbalised sampling (Zhang et al.) | Asks the model for several responses with stated probabilities instead of one. | It works through a prompt, so it is usable here. The paper measures diversity in poems, stories and jokes. It does not measure reader preference for non-fiction prose. |
| Writing quality reward model (Chakrabarty, Laban and Wu) | A ranker trained on 1,282 pairs of model paragraphs and expert edits. It scored 74% on a benchmark of 4,729 writing judgements where most frontier models were near chance. Nine experienced writers preferred its pick 66% of the time. | The weights are open, so it is a candidate component. The training data is paragraph-level and leans towards creative writing, and 66% leaves a third of picks against the experts. |

All three rewrite tools act on a finished draft. A rewrite pass can change how a sentence is built, and it cannot supply a fact, a number or a claim that the draft never had. The humanizer skill handles this by asking the writer for the missing detail, which moves the work back before the draft.

## 4. The three failure claims

The evidence found agrees with all three claims in the brief, and none of the three has a controlled study behind it.

| Claim | What was found |
| --- | --- |
| Banning a pattern produces a substitute in the same position | Two of the three rewrite tools write the substitute into the rule. One practitioner report scored outputs on nine markers and saw em dash density rise when vocabulary, copula and hedging rules were applied. No controlled measurement was found. |
| Positive style rules produce new tells | Both stop-slop and humanizer carry a rule to vary sentence length. No study was found that measures what that rule does to the sequence of sentence lengths. Section 5 gives a human reference for that sequence. |
| A draft free of listed patterns still reads as machine-written | Supported. In Shaib et al. the strongest predictors of a slop label were relevance and density, which no pattern list covers. In Russell et al. the articles that models had been prompted to humanise were still caught by the expert vote. |

### Where the findings disagree with the brief

1. The goal names a careful reader, and the result depends on which reader that is. In Russell et al. people who rarely use language models detected poorly and gave high confidence scores. Five frequent users voting together missed 1 article in 300. View: going undetected by frequent users is a harder target than being preferred, and no prompt-only method was found that has met it.
2. Hypothesis 2 puts the cause in next-token prediction. Section 2 shows the evidence puts it in post-training.
3. The brief calls detectors weak and inconsistent. In Chakrabarty et al. the leading detectors flagged 97% of prompted outputs and 3% of fine-tuned outputs, and the fine-tuned outputs were the ones readers preferred. The brief's decision not to use detectors as the measure still stands, because the target is readers. View: a strong detector is usable as a cheap regression alarm between rounds of human reading. The owner has not accepted this view.
4. The brief treats human readers as one measure. In Shaib et al. three professional copy-editors agreed on the binary slop label at Cohen's kappa of -0.15, 0.29 and 0.06 across the three pairs. A yes or no from one reader is a noisy measure, so section 6 uses several readers and paired comparisons.

## 5. Experiments

One measurement was run, on human text. No model output was generated or compared, so step 2 of the brief is still open.

### Why no generation was run

The sandbox for that session had no API key. Text written inside that conversation had the brief, the list of tells and the hypotheses in context. It was not a baseline and not a fair test of any condition, so none was presented as evidence.

### What was run: a human reference profile

357 non-fiction documents from the Brown corpus were measured, 724,854 words across eight genres from news to learned writing. The corpus is American English published in 1961, so the figures are a reference for edited human prose and not for current web writing. The script is `scripts/brown_reference.py` and its output is `reference/brown_reference.json`.

| Measure | Pooled | 10th percentile of documents | Median | 90th percentile |
| --- | --- | --- | --- | --- |
| Dashes per 1,000 words | 3.1 | 0.0 | 2.2 | 7.4 |
| Dashes, semicolons, colons and opening parentheses per 1,000 words | 10.8 | 3.2 | 8.9 | 20.3 |
| Lag-1 autocorrelation of sentence length | 0.15 | -0.11 | 0.05 | 0.22 |

Three things follow from the table.

- 52 of the 357 documents contain no dash. A text with no dashes is ordinary human prose, so the tell in claim 1 is the substitute mark and not the absence.
- The combined rate of clause-joining marks gives a way to detect substitution without listing each substitute. If a ban removes dashes and the combined rate does not fall, the marks have moved.
- Sentence length in these documents has a mean of 20.0 words and a coefficient of variation of 0.61. Neighbouring sentences are weakly similar in length when pooled. A short sentence paid off by a long one, repeated, would push the autocorrelation below zero. A single 2,000-word document is too short to show this, as 121 of the 357 human documents are below zero.

### Experiments queued

See `EXPERIMENTS.md` for the full specification of the eight experiments.

Human text on the same brief exists in public data. Russell et al. (github.com/jenna-russell/human_detectors) pair each model article with the human article that carried the same headline, and Shaib et al. (github.com/cshaib/slop) add span annotations to part of that set.

## 6. Evaluation method

The proposed measure is blind paired comparison by several human readers, with a model judge admitted only after it has been checked against those readers. This is a proposal and has not been run. See `EVALUATION.md`.

## 7. Design direction

View: the findings support a pipeline run in an agent harness, with a versioned evaluation set and a measurement script, and they do not support a single prompt or a rewrite skill. Every stage below is a hypothesis until the experiments have run.

### Stages

1. Material. Before any drafting, the system collects what the piece will rest on: the claim it commits to, the names, the numbers and at least one thing the owner saw or measured. Where material is missing it asks or writes a shorter piece. This follows from hypothesis 3 and from the finding that relevance and density drive slop labels. Experiment 4 tests it.
2. Voice. The voice is given as human-written passages in that voice, with a short specification beside them. Reinhart et al. found that tuned models kept a noun-heavy style when prompted to match informal speech, so a specification alone is expected to be the weakest lever. Experiment 6 tests it.
3. Candidates. The system produces several drafts from differently framed requests and keeps the full set. This is the usable part of verbalised sampling. Experiment 7 tests whether selection from a set beats one draft.
4. Reader pass. A second agent in a fresh context reads the draft without the brief or the voice rules. It reports where it stopped believing the text and what it wanted to know that the text did not say. Revision deals with content before sentences. Experiment 5 tests it.
5. Measurement. The profile script runs on the output, and the output is logged with the versions of the prompts, voice file and model that produced it.

### Why a pipeline and not a skill

A skill that rewrites a draft sits after the point where the main fault enters. Three of the five stages need separate contexts: the candidates, the reader and the judge. The evaluation set and the logs need files and scripts that persist between runs. Claude Code supplies all of these through subagents, skills and a repository.

### Feedback and change control

- Feedback is filed in one of two places. A quality ledger holds faults a reader would call bad writing in any voice. A file per voice holds preferences that belong to that voice.
- Each item is stored with the passage that prompted it and the brief it came from. The system proposes the classification and the owner confirms it.
- An item becomes a rule only after it recurs across at least two briefs or two readers. This is the guard against the most recent criticism.
- A quality rule has to hold in at least two voices on the held-out set before it is accepted. This is the guard against the owner's own taste entering the quality ledger.
- Every accepted change is run on the 24 held-out briefs. The profile is compared with the previous run for any measure that moved, and readers see a sample of pairs before the change is kept.
- Rules can be retired, and each rule records the evidence that admitted it.

## 8. Assumptions, limits and open questions

Nothing in this document has been tested on model output, and the design in section 7 could change once it has.

### Assumptions

- The prose in scope is English non-fiction of roughly 300 to 2,000 words.
- The generator is a current Claude model reached through the API or Claude Code.
- The owner can supply material for each piece and can recruit at least four other readers.

### Known limits

- The studies cited used GPT-4o, Claude 3.5 Sonnet, o1 and Llama 3. Current models may differ, and experiment 1 is the check.
- The system cannot supply an observation it has not been given. Where there is no material, the output will be general or short.
- Going undetected by frequent users of language models may be out of reach without fine-tuning. The one result found where expert readers preferred model text used fine-tuning.
- The human reference in section 5 is from 1961. A reference from current writing in each target genre has to replace it before the profile is used as an alarm.
- The test voice specification asks for an even register with rare parallelism. A voice defined by what is kept rare may produce its own outlier on the profile, and experiment 8 looks for it.

## Sources

- Reinhart et al., Do LLMs write like humans? Variation in grammatical and rhetorical styles. https://arxiv.org/abs/2410.16107
- Russell, Karpinska and Iyyer, People who frequently use ChatGPT for writing tasks are accurate and robust detectors of AI-generated text. https://arxiv.org/abs/2501.15654
- Zhang et al., Verbalized Sampling: How to Mitigate Mode Collapse and Unlock LLM Diversity. https://arxiv.org/abs/2510.01171v3
- Shaib et al., Measuring AI "Slop" in Text. https://arxiv.org/abs/2509.19163v2
- Chakrabarty, Ginsburg and Dhillon, Readers Prefer Outputs of AI Trained on Copyrighted Books over Expert Human Writers. https://arxiv.org/abs/2510.13939v4
- Chakrabarty, Laban and Wu, AI-Slop to AI-Polish? https://arxiv.org/abs/2504.07532v3
- Paech et al., Antislop. https://arxiv.org/abs/2510.15061
- blader/humanizer, SKILL.md on the main branch, read on 11 November 2026. https://github.com/blader/humanizer
- stop-slop by Hardik Pandya, the copy installed in the owner's plugins
- de-slopify prompt. https://playbooks.com/skills/dicklesworthstone/agent_flywheel_clawdbot_skills_and_integrations/de-slopify
- Practitioner report on nine markers. https://reddit.sentinel-team.org/posts/1s8wf8i/snapshots/2026-04-03T23%3A00%3A15.436463Z
- Brown corpus, through the NLTK distribution
