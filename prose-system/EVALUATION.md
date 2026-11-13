# Evaluation method

This is a proposal from the investigation. It has not been run. The measure of success in the brief is human readers, so the primary measure is blind paired comparison by several readers. A model judge is admitted only after it has been checked against those readers.

## Human reader protocol

- Each trial shows two passages written to the same brief, in random order and with no source labels.
- The reader picks the passage they would rather read, marks each passage as human or model, and gives one line of reason.
- Three pairings are used: system against baseline, system against human, and baseline against human. The third shows how far the baseline already is from the human text.
- At least five readers, including the owner. Russell et al. used a vote of five.
- Record how often each reader uses language models. In Russell et al. that predicted detection skill.
- 30 judgements per pairing is the minimum. A preference of 21 in 30 differs from an even split at p = 0.043, two-sided. A preference of 15 in 20 gives p = 0.041, and 27 in 40 gives p = 0.038. Smaller differences need more pairs.
- Preference and detection are reported as two numbers. A passage can be preferred and still be identified.
- Report agreement between readers. In Shaib et al. three professional copy-editors agreed on a binary slop label at Cohen's kappa of -0.15, 0.29 and 0.06, so low agreement on a yes or no label is expected and is the reason for paired comparisons.

The instrument has to work for readers who are not engineers. A web page that shows one pair at a time and stores the answers is enough. Readers must not see condition names, file names or any ordering that reveals the source.

## Briefs

- 24 held-out briefs across four kinds of prose: explanation, argument, personal or narrative non-fiction, and technical description. Each has a human reference text where one exists.
- 12 development briefs for tuning.
- The owner approves both sets, and then they are frozen and versioned.
- The held-out set is never used to write or adjust a prompt. If a held-out brief is used for tuning by mistake, replace it and record the replacement.

## Baseline

- One named model version with one plain prompt per brief, sampled 10 times and stored.
- Every later result is compared with those stored samples. If the model version changes, the baseline is rerun and both are kept.

## Model judge

The published record on uncalibrated model judges is poor.

- In Shaib et al., GPT-5, DeepSeek-V3 and o3-mini agreed with human slop labels at kappa between -0.01 and 0.03. They labelled 3% to 8% of texts as slop where the humans labelled 34%.
- In Chakrabarty, Laban and Wu, most frontier models were near chance on 4,729 writing-quality judgements.

Calibration procedure:

1. Collect the human paired judgements first.
2. Run the judge on the same pairs, in both orders, in fresh contexts.
3. Report the judge's agreement with the reader majority next to the readers' agreement with each other.
4. List the pairs where the judge disagrees with the majority, with the passages.
5. Use the judge for screening between human rounds only if its agreement is close to the readers' own. It does not replace a human round before a change is accepted.

Where possible, use a judge from a different model family from the generator, and test whether the judge favours text from its own family.

## Automated profile

`scripts/profile.py` reports marks per 1,000 words, sentence length statistics, the short-then-long share, numbers and capitalised names per 100 words, and phrases repeated across samples. Its job is to raise an alarm when a change moves a mark or a rhythm, including a substitute that no list names. It is not a quality score, and no change is accepted on the profile alone.

The human reference in `reference/brown_reference.json` is from 1961 and was computed on pre-tokenised text. Build a current reference by running `scripts/profile.py` on human text in each target genre before using the profile as an alarm.

## Detectors

The brief says not to use AI-text detectors as the measure. The findings suggest a strong detector could serve as a regression alarm between human rounds. The owner has not accepted that suggestion. Ask before using one.
