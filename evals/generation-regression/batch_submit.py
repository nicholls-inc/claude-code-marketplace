"""Assemble one-shot patch-generation requests for the generation eval and submit as a batch."""
import json
import pathlib
import subprocess
import sys

import anthropic

GEN = pathlib.Path(__file__).parent
CORE = "/Users/harry.nicholls/repos/core"
MODEL = "claude-opus-5"
K = 3

CASES = {
    16635: {"target_files": ["ocpp_kafka/ocpp_kafka_consumer.py", "shared/kafka/base_kafka_consumer.py", "inventev/settings/base.py", "inventev/settings/test.py", "ocpp_kafka/tests/test_ocpp_kafka_consumer.py", "shared/tests/kafka/test_base_kafka_consumer.py"]},
    13705: {"target_files": ["vehicle_onboarding/views/generic.py", "vehicle_onboarding/forms.py", "vehicle_onboarding/urls.py", "vehicle_onboarding/templates/vehicle_onboarding/make_selection.html", "vehicle_onboarding/tests/test_webflow.py", "dc/models/car.py"]},
    12762: {"target_files": ["dc/third_party_api_clients/gridx/gridx_api_client.py", "dc/third_party_api_clients/gridx/payloads.py", "dc/tasks/multi_queue_huey/__init__.py", "inventev/settings/base.py", "dc/logic/openadr_3/payloads.py"]},
    5326: {"target_files": ["dc/controllers/vehicles/smartcar.py", "dc/models/vehicle_api_log.py", "dc/models/__init__.py", "dc/tests/controllers/vehicles/test_smartcar.py"]},
    4614: {"target_files": ["dc/views/rest/stripe.py", "dc/tasks/payment.py", "dc/enums/enums.py", "dc/utils/converters.py", "dc/admin/payment.py"]},
    9688: {"target_files": ["api_v2/views/notifications/notifications.py", "api_v2/views/base.py", "dc/models/organisation.py", "dc/models/application_extra.py", "dc/models/notification.py"]},
    10279: {"target_files": ["dc/models/solar.py", "dc/data/combined_prices.py", "dc/scheduler/solar_only_schedules.py", "dc/scheduler/solar_and_grid_schedules.py", "dc/scheduler/solar_smart_schedules.py", "dc/models/scheduleabledevice.py"]},
}

DISCIPLINE = {
    "bare": "Implement the change directly.",
    "delib": "Before writing any code, write a short design: restate the requirements in your own words, describe your planned approach and the data flow, and list the decisions you are making. Then implement according to that design.",
    "cc": "Before writing any code: (1) enumerate the invariants and failure modes of the component you are changing — boundary conditions, empty inputs, type/relation mismatches, unit errors, error paths; write them as an explicit numbered contract. (2) Implement. (3) Before finishing, re-audit your diff line by line against each invariant and state for each how the code satisfies it, fixing anything that does not. This contract-first discipline is mandatory.",
}

MAX_FILE_CHARS = 60_000


def git_show(sha: str, path: str) -> str | None:
    r = subprocess.run(["git", "-C", CORE, "show", f"{sha}:{path}"], capture_output=True, text=True)
    return r.stdout if r.returncode == 0 else None


def build_prompt(pr: int, arm: str, parents: dict) -> str:
    spec = (GEN / f"spec_{pr}.md").read_text()
    sha = parents[pr]
    parts = [
        DISCIPLINE[arm],
        "\nTask specification:\n\n" + spec,
        "\nBelow are the current contents of the relevant files at the commit you are working from. A file marked (does not exist yet) must be created.\n",
    ]
    for path in CASES[pr]["target_files"]:
        content = git_show(sha, path)
        if content is None:
            parts.append(f"=== {path} (does not exist yet) ===\n")
        else:
            if len(content) > MAX_FILE_CHARS:
                content = content[:MAX_FILE_CHARS] + "\n# ... [truncated]\n"
            parts.append(f"=== {path} ===\n{content}\n")
    parts.append(
        "\nOutput requirements: respond with a single unified diff (git format, `--- a/path` / `+++ b/path` headers, `/dev/null` for new files) implementing the task across these files, and nothing else after the diff. Any design/invariant analysis required by your instructions comes BEFORE the diff. The diff must be syntactically valid Python/Django code."
    )
    return "\n".join(parts)


def main() -> None:
    parents = dict(
        line.split("\t") for line in (GEN / "parents.tsv").read_text().splitlines()
    )
    parents = {int(k): v for k, v in parents.items()}
    requests = []
    for pr in CASES:
        for arm in DISCIPLINE:
            prompt = build_prompt(pr, arm, parents)
            for k in range(1, K + 1):
                requests.append(
                    {
                        "custom_id": f"{pr}_{arm}_r{k}",
                        "params": {
                            "model": MODEL,
                            "max_tokens": 64000,
                            "messages": [{"role": "user", "content": prompt}],
                        },
                    }
                )
    sizes = sorted(len(r["params"]["messages"][0]["content"]) for r in requests)
    print(f"requests: {len(requests)}, prompt chars min/median/max: {sizes[0]}/{sizes[len(sizes)//2]}/{sizes[-1]}")
    if "--dry-run" in sys.argv:
        return
    client = anthropic.Anthropic()
    batch = client.messages.batches.create(requests=requests)
    (GEN / "batch_id.txt").write_text(batch.id)
    print("batch:", batch.id, batch.processing_status)


if __name__ == "__main__":
    main()
