# prose

Two skills for writing non-fiction with a language model so that it sits closer to skilled human prose, and for checking any text against a measured human reference.

- `/prose:write` collects the material, drafts five candidates in fresh contexts under a voice file, ranks them against a reference of human text, checks the pick's numbers and names against the material, and returns the pick without a style rewrite.
- `/prose:check` measures a text and reports where it falls outside the human range. It reports and does not rewrite.

The procedure follows measurements on 1,848 generated samples and 133 human documents. [docs/evidence.md](docs/evidence.md) gives the finding behind each step. No reader study has been run, so the claim is that the output is closer to the human profile on surface measures, not that readers prefer it.

## Install

```
/plugin marketplace add nicholls-inc/claude-code-marketplace
/plugin install prose@nicholls
```

## Use

```
/prose:write a 600-word explanation of our new on-call rota for the engineering team
/prose:check docs/announcement.md --kind explanation
```

`/prose:write` asks for anything it needs: the reader, the length, and the facts the piece will rest on. It will not supply facts from memory. Give it names, numbers and anything you saw first-hand, or point it at a file that holds them.

The script also runs on its own, with Python 3 and no dependencies:

```bash
python3 scripts/prose_profile.py check draft.md --kind argument
python3 scripts/prose_profile.py rank drafts/*.txt --kind narrative
python3 scripts/prose_profile.py specifics draft.md --material notes.md
```

`--kind` is `explanation`, `argument`, `narrative` or `technical`.

## What the measures are

| Measure | What it catches |
| --- | --- |
| Mean sentence length and its spread | Choppy text, and text pushed into very long sentences by a rule |
| Sentences of four words or fewer | Short hooks such as "It did not work." |
| Short sentence then a much longer one | The hook-and-payoff pattern. The script quotes the pairs it finds. |
| Lag-1 autocorrelation of sentence length | Regular alternation of short and long sentences. Human text sits slightly above zero and model text below it. |
| Commas, dashes, semicolons, colons, parentheses per 1,000 words | A mark that has gone missing or piled up |
| Numbers and names per 100 words | How much concrete material the text carries |

Each value is shown beside the 10th to 90th percentile of the human reference. One human document in five is outside each range, so a single flag means little.

## Voices

A voice is a short description of a register in a text file. `voices/plain.md` is the only voice tested so far. To add one, put a file in your project at `.prose/voices/<name>.md` and pass `--voice <name>`. Describe the register as a whole. Do not write a voice as a list of banned patterns or as a single rule about rhythm, because both made the text worse in testing.

## Feedback

`/prose:write` records comments in the project and keeps two kinds apart: `.prose/quality-ledger.md` for faults that are bad writing in any voice, and notes beside each voice file for preferences that belong to that voice. A comment changes a voice file only after it recurs on a second brief and you approve it, and the change is compared on three earlier briefs first.

## Known limits

- The pick is chosen by the same measures it is judged on. Whether readers prefer it has not been tested.
- The measures are surface features. A text can pass every one and still be vague or wrong.
- The specifics check compares digits and capitalised names with the material. It misses spelled-out numbers and reworded claims.
- Tested on English non-fiction of 300 to 2,000 words, mostly with Claude Sonnet 5.5 as the drafting model.
- Drafting five candidates costs about five times one draft, plus the fixed context of each subagent.
- The skills have been run by hand, step by step, in a session. They have not yet been run as an installed plugin.

## Example

[docs/example](docs/example) holds one run: the material file, the ranking of five candidates and the chosen piece.
