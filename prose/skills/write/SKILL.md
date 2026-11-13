---
name: write
description: >-
  Write a piece of non-fiction prose that sits closer to skilled human writing
  than a single model draft does. Collects the material first, drafts several
  candidates in fresh contexts under a voice file, measures each against a
  reference of human text and picks by that measure. Use for articles, essays,
  explanations, posts and reports of roughly 300 to 2,000 words. Triggers:
  "write this up", "draft an article", "write a post about", "prose write".
argument-hint: "<what to write> [--voice <name or path>] [--kind explanation|argument|narrative|technical] [--candidates N]"
---
# /prose:write

Produce one piece of prose by drafting several candidates and choosing among them by measurement. Do not write the piece yourself in this conversation, and do not polish the chosen draft's style afterwards. Both steps were tested and both made the text worse on the measures. The reasons are in [the evidence](${CLAUDE_PLUGIN_ROOT}/docs/evidence.md).

## Steps

### 1. Settle the brief

You need four things. Ask for any that are missing and cannot be inferred.

- What the piece is about and what it should leave the reader knowing or believing.
- Who the reader is.
- The length in words.
- The kind: explanation, argument, narrative or technical. This picks the reference the drafts are measured against.

### 2. Collect the material

Write a material file before any drafting. It holds what the piece will rest on:

- the claim or position the piece commits to;
- the names, numbers, dates and quotations it may use, each as the user gave it or as you read it in a source the user pointed to;
- anything the user saw, measured or did first-hand.

Do not add facts from memory to the material file. If the piece needs a fact the user has not supplied and no source is at hand, ask for it. If the user has nothing to add, tell them the piece will be more general or shorter, and proceed.

Save it as `material.md` in a new working folder under the scratchpad or a temporary directory.

### 3. Choose the voice

Default to `${CLAUDE_PLUGIN_ROOT}/voices/plain.md`. If the user names another voice, look for it in the project's `.prose/voices/` folder, then in `${CLAUDE_PLUGIN_ROOT}/voices/`. A voice file is a short description of a register. Use its text as written.

### 4. Draft the candidates

Start five subagents at once, each in a fresh context, with the Agent tool. Use more only if the user asks. Do not use a Haiku model for drafting: in testing its subagents called unrelated tools by mistake.

Give every subagent the same prompt and nothing else. Build it from these parts in this order:

1. The brief as one or two plain sentences: what to write, the length, the reader.
2. `Use only the facts, names, numbers and quotations in the material below. Do not add others.` followed by the material file's text.
3. `Write it as plain prose with no headings or lists.` Leave this out only if the user asked for headings or lists.
4. `Write it in this voice:` followed by the voice file's text.
5. `Save the piece to <working folder>/c<N>.txt and reply with the single word done.` with a different N for each subagent.

Do not add anything else to the prompt. In particular, do not add:

- a rule about sentence rhythm, such as "vary your sentence length" or "avoid short punchy sentences";
- a ban on a punctuation mark or a list of phrases to avoid;
- this conversation's history, the user's feedback on earlier drafts, or examples of bad writing.

### 5. Measure and pick

First confirm that every candidate file exists. A subagent can finish without saving its file. Start a replacement for each missing one, and do not rank fewer than three candidates.

```bash
python3 "${CLAUDE_PLUGIN_ROOT}/scripts/prose_profile.py" rank <working folder>/c*.txt --kind <kind>
```

Take the first-ranked candidate. Then check its specifics against the material:

```bash
python3 "${CLAUDE_PLUGIN_ROOT}/scripts/prose_profile.py" specifics <working folder>/c<N>.txt --material <working folder>/material.md
```

Read every item the check lists. If the candidate uses a number, name or quotation that the material does not support, move to the next-ranked candidate and check that one. If every candidate has unsupported specifics, take the best-ranked one and remove or correct the unsupported sentences. That is a content edit and is allowed.

### 6. Leave the style alone

Do not revise the chosen draft for rhythm, word choice or punctuation. The edits you may make are:

- correcting or removing a fact the material does not support;
- a change the user asks for.

If the chosen draft is outside the human range on a measure, say so in the report. If the user wants a better one, draft five more candidates and rank all ten. Do not fix a measure by editing sentences.

### 7. Deliver

Give the user the piece exactly as chosen, then a short report:

- how many candidates were drafted and which was picked;
- the measures on which the pick is outside the human range, if any;
- any specifics you removed or corrected, and any you could not verify;
- one line saying the measures track surface features and are not a judgement of quality.

## Feedback

When the user comments on a piece, decide with them whether the comment is about quality or about voice, and record it in the project:

- `.prose/quality-ledger.md` for a fault a reader would call bad writing in any voice;
- `.prose/voices/<voice>.md` notes for a preference that belongs to one voice.

Record the passage that prompted the comment and the brief it came from. Do not change a voice file or this procedure on a single comment. A comment becomes a change only after it recurs on a second brief, and the user approves the change. Before keeping a change to a voice file, draft with the old and new file on three earlier briefs and compare the ranked measures, so that a fix for one fault does not introduce another.
