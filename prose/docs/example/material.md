Claim: on current Claude models, the usual advice for making AI writing sound human targets the wrong things, and single style rules make the text worse on measurable features.

Facts that may be used:
- The test covered 1,848 generated samples on 12 briefs, 10 samples per brief per condition, in November 2026.
- Most samples came from Claude Sonnet 5.5. Baselines were also taken on Opus 5.5, Fable 5.2 and Haiku 5.5.
- The human reference was 133 documents, about 258,000 words, published before November 2022.
- The Sonnet 5.5 baseline contained no em dash in 120 samples, about 109,000 words. The human texts used 2.9 dashes per 1,000 words.
- Baseline mean sentence length was 14.6 words. The human mean was 20.3 words.
- 57% of baseline samples used headings and 36% used lists.
- With an instruction to write plain prose with no headings or lists, mean sentence length was 19.0 words.
- The instruction "Vary your sentence length" raised the share of sentences of four words or fewer from 7.3% to 10.1%.
- The instruction "Do not use a short sentence to set up a longer one" raised mean sentence length to 39.6 words.
- Banning dashes, semicolons, colons and parentheses cut digits per 100 words from 2.2 to 1.2.
- A revision pass asked to fix rhythm raised mean sentence length from 19.0 to 27.0 words and raised colons, semicolons and commas.
- In every model and condition, short and long sentences alternated more than in the human texts. The lag-1 autocorrelation of sentence length was between -0.07 and -0.19 for the models and 0.04 for the human texts.
- Drafting five candidates under a full voice description and picking the one closest to the human profile moved the autocorrelation from -0.14 to -0.05.
- No reader study was run. One reader judged six pairs in a pilot. All measures are surface features counted by a script.
