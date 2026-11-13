---
name: check
description: >-
  Measure a piece of prose against a reference of human-written text and report
  where it sits outside the human range: sentence length, short sentences set up
  to be paid off by long ones, alternation of sentence length, punctuation,
  numbers and names. Reports and explains; does not rewrite. Triggers: "check
  this draft", "does this read like AI", "prose check", "profile this text".
argument-hint: "<file or pasted text> [--kind explanation|argument|narrative|technical]"
---
# /prose:check

Measure a text and report what the measurement shows. Do not rewrite the text unless the user asks.

## Steps

1. Get the text into a file. If the user pasted it, save it unchanged to a temporary file.
2. Run the check. Add `--kind` if the user says what kind of prose it is or it is obvious.

   ```bash
   python3 "${CLAUDE_PLUGIN_ROOT}/scripts/prose_profile.py" check <file> --kind <kind>
   ```

3. Report in plain words:
   - the measures flagged LOW or HIGH, each with its value and the human range;
   - the short-then-long sentence pairs the script lists, quoted;
   - headings and lists, if the count is not zero, since the reference is running prose.
4. Say what the result does and does not mean:
   - The range is the 10th to 90th percentile of 133 human documents, so one human document in five is outside each range. One flag is weak evidence. Several flags on the rhythm measures together are stronger.
   - The measures track surface features. A text inside every range can still be vague or wrong, and the script cannot tell.
   - A text under 300 words is too short for the rhythm measures.

## If the user asks for a fix

Do not edit sentences to move a measure. In testing, a revision pass asked to fix rhythm raised the mean sentence length from 19.0 to 27.0 words and added colons, semicolons and commas where it joined sentences. Offer these instead:

- **Formatting flags.** If headings and lists are the cause, ask whether the piece needs them. Rewriting as plain prose is a structural change the user can approve.
- **Rhythm flags.** Offer to produce a new version with `/prose:write`, using the text's content as the material. That drafts several candidates and picks by measurement.
- **Few numbers or names.** Ask the user for the specifics. Do not supply them from memory.

The reasons are in [the evidence](${CLAUDE_PLUGIN_ROOT}/docs/evidence.md).
