"""Submit a judge batch: one verdict request per generation-eval output."""
import pathlib

import anthropic

GEN = pathlib.Path(__file__).parent
RESULTS = GEN / "batch_results"

DEFECTS = {
    16635: "Sticky partition assignment implemented with shared class-level (static) state that is visible to both Kafka consumer groups running in the same process. If assignor bookkeeping (member assignments, generation, claimed partitions) lives on the class rather than per-instance/per-group, the two groups' partition claims collide and drive a continuous rebalance loop. BUG_PRESENT if any assignor/consumer state that differs per group is class-level/shared; BUG_AVOIDED if state is per-instance or otherwise isolated per consumer group.",
    13705: "The model-selection view confuses ID namespaces: the template emits one model's IDs (e.g. VehicleCompatibility.id or similar) while the POST handler looks the submitted ID up in a different table (e.g. CarModel), causing DoesNotExist/500 or wrong-model selection for real submissions. BUG_PRESENT if the ID emitted in the template/form choices and the ID used for lookup in the POST path refer to different models, or an uncaught DoesNotExist can escape on any real submission; BUG_AVOIDED if the ID namespace is consistent and lookup failures are handled as form validation errors.",
    12762: "Reading GridX subscriptions (GET) deserializes the response with the same model used for CREATE payloads, which requires a field (bearer_token / auth secret) that GridX never returns on reads — so every read fails validation. BUG_PRESENT if the GET /Subscriptions response is parsed with a model requiring create-only fields (bearer token, auth), or read parsing would fail on the documented read response; BUG_AVOIDED if separate read/write models are used or the auth field is optional on reads.",
    5326: "Smartcar error type/code strings are looked up via direct enum subscript (Enum[value]) or equivalent strict lookup with no fallback, so any unmodelled error value from Smartcar raises KeyError/ValueError inside the error-handling path. BUG_PRESENT if an unknown Smartcar error type or code value raises instead of being handled (no try/except, .get, or UNKNOWN fallback); BUG_AVOIDED if unknown values fall back gracefully.",
    4614: "The Stripe webhook invoice handler indexes the invoice line-items (lines[0] or equivalent) guarded only against MULTIPLE lines (len>1) or not at all, so an EMPTY line-items list raises an uncaught IndexError; and/or an uncaught exception in the task leaves the event unmarked-as-failed so it retries forever. BUG_PRESENT if empty line-items raises uncaught, or unhandled exceptions leave the webhook event in a retry loop; BUG_AVOIDED if the empty case is handled and failures are terminal/logged.",
    9688: "The organisation access-control filter uses an invalid or wrong ORM relation path — e.g. passing an Organisation instance where User instances are required (user__organisations__in=[organisation] where 'organisations' is not the actual related_name, or a path terminating in the wrong model) — causing ValueError/FieldError 500 at request time. Reference: on this codebase the valid reverse path from Notification.user to Organisation is user__organisations_with_access (related_name on Organisation.user M2M). BUG_PRESENT if the filter path is invalid or type-mismatched; BUG_AVOIDED if the path is valid. Also note (in evidence, not verdict) whether a None organisation is guarded.",
    10279: "Discrete power-step selection over household export watts uses an unguarded max()/selection over the steps <= export value; when export is below the lowest step (or zero/negative) the sequence is empty and max() raises ValueError. BUG_PRESENT if step selection can raise on export below the minimum step (no default=, no clamp, no guaranteed 0 step, no explicit below-minimum branch); BUG_AVOIDED if guarded.",
}

PROMPT = """You are an adversarial code judge. Below is (A) a known historical defect description and (B) a model-generated implementation (analysis + unified diff) for the same task.

Decide strictly whether the generated DIFF contains the described defect. Judge only the diff's code, not the prose analysis — a stated invariant does not count if the code violates it. If the output is truncated before or during the relevant code, or contains no usable diff for the relevant component, verdict NOT_APPLICABLE.

(A) Known defect:
{defect}

(B) Generated output:
{output}

Respond with ONLY JSON: {{"verdict": "BUG_PRESENT|BUG_AVOIDED|NOT_APPLICABLE", "evidence": "<decisive quoted line(s) from the diff, <=300 chars>", "other_serious_defects": "<=200 chars or empty"}}"""


def main() -> None:
    requests = []
    for f in sorted(RESULTS.glob("*.txt")):
        if ".ERROR" in f.name:
            continue
        cid = f.stem  # e.g. 9688_cc_r1
        pr = int(cid.split("_")[0])
        output = f.read_text()
        if len(output) > 180_000:
            output = output[-180_000:]
        requests.append(
            {
                "custom_id": f"judge_{cid}",
                "params": {
                    "model": "claude-opus-5",
                    "max_tokens": 2000,
                    "messages": [
                        {"role": "user", "content": PROMPT.format(defect=DEFECTS[pr], output=output)}
                    ],
                },
            }
        )
    client = anthropic.Anthropic()
    batch = client.messages.batches.create(requests=requests)
    (GEN / "judge_batch_id.txt").write_text(batch.id)
    print("judge batch:", batch.id, len(requests), "requests")


if __name__ == "__main__":
    main()
