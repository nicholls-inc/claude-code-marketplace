# Brief (owner's original, unchanged)

Task: build a system for producing AI-generated prose that reads as if a skilled human wrote it

## Background

I'm a senior product engineer. I use frontier language models through prompting and tooling, including agent harnesses such as Claude Code. I will not train or fine-tune models, so the system has to work with models as they are available through APIs and agent tools.

AI-generated prose has recognisable failings. Readers spot it within a few sentences, and once they do, they trust the content less. My current understanding of the causes is as follows. Preference tuning rewards text that looks finished to a rater who is skimming, which narrows output towards one register. Next-token prediction favours the most likely phrase, which is often a stock one. The model has no observation of its own to draw on, so it fills space with general claims. It also writes in a single pass and doesn't reread its draft as a reader would. Treat these as hypotheses to test, not as settled facts.

## What hasn't worked

Most existing tools catalogue surface patterns such as em dashes, "it's not X, it's Y" contrasts, stock phrases, quotable paragraph endings and lists of three. They then tell the model to avoid those patterns or rewrite them out. These tools have not worked for me, and I have seen three reasons why:

- Banning a pattern produces a substitute in the same position. Remove em dashes and semicolons or colons appear where the dashes were.
- Positive style rules produce new tells. An instruction to vary sentence length tends to produce a short sentence followed by a longer one that pays it off, repeated down the paragraph.
- A draft can be free of every listed pattern and still read as machine-written, because its claims are vague, it lacks specifics, and it never commits to a point.

Start beyond this point rather than reproducing it. If your investigation shows I'm wrong about any of this, say so and show the evidence.

## Goal

Build a system I can use to produce prose that a careful reader would not identify as AI-generated, and that reads as good writing on its own terms. Quality matters more to me than voice. I also want control over voice: I should be able to specify a voice and get it, and changing the voice should not lower the quality or introduce new tells. The voice does not need to be mine.

I mean "system" in the general sense. It could be a set of prompts and procedures, a pipeline of agents, a piece of software, a written method, or a combination of these. Choose the form your findings support and explain why you chose it.

The measure of success is human readers. Do not use AI-text detectors as the measure. They are weak and inconsistent, and beating them is a different problem from writing well.

## How to work

1. Investigate before designing. Survey existing tools and prompts. Look at research on LLM prose quality, mode collapse and stylometry, and at work on what distinguishes skilled human prose. For each existing approach, find out where it falls short and collect examples.
2. Run experiments. Generate text, compare it with human-written text on the same brief, and record what you find. Any claim about what works should rest on outputs you produced and examined.
3. Set up an evaluation method before building the system, so you can measure whether each change helps. Explain how you know the method tracks what a human reader would judge. If you use a model as judge, show how you checked it against human judgement and where the two disagree.
4. Keep a baseline. The baseline is plain prompting of a current frontier model with a reasonable brief. Measure every claimed improvement against it.

Write up the investigation findings before you build anything large, and let them drive the design. State your assumptions as you go.

## Evolution and feedback

The system will need to change as I use it and give feedback. Design for that, with two safeguards:

- Feedback should not pull the system towards my personal tastes or towards whatever was criticised most recently. Keep feedback about quality separate from feedback about voice.
- Check each change against a fixed, held-out set of briefs, so that fixing one problem doesn't cause another or create a substitute tell.

## Voice specification for testing

As one test of voice control, use the following specification, which a reader wrote for their preferred register. Treat it as one input among several, not as the definition of good writing.

> Calm, even register. Mostly medium-length, complete sentences, with length set by content rather than rhythm. No pattern of a short hook paid off by a longer sentence. Plain words over writerly ones, and lines allowed to sit flat. Numbers, names and concrete facts carry the point instead of abstractions. Em dashes, parentheticals, adverbs, doublets and parallelism kept rare. Descriptive writing states what a thing is and does, and any view is labelled as a view. British spelling.

## Deliverables

- Findings on what causes the problem and where existing approaches fail, with examples you generated.
- The system, in whatever form you chose, with instructions for using it and for feeding back into it.
- The evaluation method and its results against the baseline, including side-by-side samples.
- Known limits: what the system doesn't fix and the conditions under which it fails.
