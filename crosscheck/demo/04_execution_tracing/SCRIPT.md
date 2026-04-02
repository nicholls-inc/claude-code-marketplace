# Demo 4: Execution Tracing — SWE-bench_Pro qutebrowser host blocking

**"The helper function exists. It's just not called."**

## The bug

A host-based content blocker uses exact hostname matching. Blocking
`example.com` doesn't block `sub.example.com`. The `widened_hostnames()`
function — which generates parent domain matches — EXISTS in the same
module but is never called by `_is_blocked()`.

There's also a second bug: the whitelist check happens AFTER the block
check instead of before it. This creates a subtle interaction when
subdomain blocking is fixed: whitelisting `cdn.example.com` while blocking
`example.com` requires the whitelist to short-circuit before widening.

Source: SWE-bench_Pro `qutebrowser__qutebrowser-c580ebf0801e5a3ecabc54f327498bb753c6d5f2`

## Setup (before the demo)

- No Docker needed
- Open Claude Code in this directory

## Step 1: Show the Passing Tests and the Bug (1 min)

```bash
pytest test_hostblock.py::TestBasicBlocking -v
pytest test_hostblock.py -v -k "subdomain or whitelist_precedence" 2>&1 | tail -20
```

> "The basic blocker works fine — exact matches are blocked, whitelisting
> works. But look at the subdomain tests: blocking 'example.com' doesn't
> block 'sub.example.com'. A user reports this as a bug."

## Step 2: Ask Vanilla Claude to Fix It (1-2 min)

```
Look at hostblock_bug.py. A user reports that blocking 'example.com' in
their blocklist doesn't block 'sub.example.com' or 'a.b.example.com'.

The failing tests are in TestSubdomainBlocking. What's the root cause
and how do you fix it?
```

**Expected vanilla response:** Claude will likely:
- Correctly identify the exact-match issue in `_is_blocked`
- Suggest adding a loop or substring check
- **Miss** that `widened_hostnames()` already exists in the same file
- **Miss** the whitelist ordering issue (the SECOND bug)
- Propose a fix that works for subdomains but introduces a regression
  where whitelisted subdomains of blocked parents get blocked

**If Claude finds both bugs:** Rare. Pivot to: "Claude found both,
but the /trace-execution certificate shows HOW it traces from the
failure through the entire call graph, revealing both issues in context."

## Step 3: Use /trace-execution (2-3 min)

```
/trace-execution "Starting from HostBlocker.filter_request(), trace what
happens when:
1. _blocked_hosts = {'example.com'}
2. request_url = 'https://sub.example.com/tracker'

Build the complete call graph and identify where the execution diverges
from the expected behavior (subdomain should be blocked)."
```

**What the audience sees — hypothesis-driven tracing:**

**Entry point:** `filter_request('https://sub.example.com/tracker')`

**Call graph:**
```
filter_request(url)
  └─ _is_blocked(url)
       ├─ config.blocking_enabled → True (line 78)
       ├─ urlparse(url).hostname → 'sub.example.com' (line 82)
       ├─ 'sub.example.com' in self._blocked_hosts → False ← DIVERGENCE
       ├─ 'sub.example.com' in self._config_blocked_hosts → False
       └─ returns False (never reaches whitelist check)
```

**Observation 1:** `_is_blocked` does exact set membership (line 86-87).
The hostname `'sub.example.com'` is not literally in the set `{'example.com'}`.

**Observation 2:** `widened_hostnames()` exists at line 31 in the SAME file.
It generates `['sub.example.com', 'example.com', 'com']`. If called,
`'example.com'` would match on the second iteration.

**Hypothesis update:** The fix is to iterate `widened_hostnames(host)` and
check each against the blocked sets.

**Observation 3:** BUT — look at the whitelist check (line 87):
```python
return (...blocked...) and not self.is_whitelisted(request_url)
```
The whitelist is checked AFTER the block decision. If we add subdomain
widening, a request to `cdn.example.com` (whitelisted) with parent
`example.com` (blocked) would:
1. Widen to find `example.com` → blocked
2. Check whitelist for `cdn.example.com` → exact match in whitelist → not blocked

This works by accident for exact whitelist matches. But the whitelist check
should happen FIRST for two reasons:
- **Performance:** whitelist is tiny, blocklist can have millions of entries
- **Correctness:** whitelist should short-circuit before any blocking logic

**Key moment:** When the trace reveals `widened_hostnames` at line 31:

> "Look at that — the function to fix this bug ALREADY EXISTS. Line 31,
> same file. It generates parent domain matches. But `_is_blocked` at
> line 86 does a raw set membership check instead of using it.
>
> The trace also found a second issue: the whitelist check is in the wrong
> position. It should short-circuit BEFORE the block check, not after.
> Without systematic tracing, you'd fix the first bug and introduce a
> regression in the whitelist behavior."

## Step 4: Show the Actual Fix (30s)

The correct fix needs two changes:

```python
def _is_blocked(self, request_url, first_party_url=None):
    if not self._config.blocking_enabled:
        return False

    # Check whitelist FIRST (short-circuit)
    if self.is_whitelisted(request_url):
        return False

    host = urlparse(request_url).hostname
    if host is None:
        return False

    # Iterate through widened hostnames to catch parent domain blocks
    for hostname in widened_hostnames(host):
        if hostname in self._blocked_hosts or hostname in self._config_blocked_hosts:
            return True

    return False
```

> "Two changes: whitelist moved up, and the existing widened_hostnames()
> function is now called. The trace found both because it built the
> complete call graph instead of fixating on the set membership line."

## Timing Budget

| Step | Duration |
|------|----------|
| Show tests | 1:00 |
| Vanilla Claude | 1:30 |
| /trace-execution | 2:30 |
| Show fix | 0:30 |
| **Total** | **~5:30** |

## Contingencies

**Claude suggests using widened_hostnames:**
"Good — Claude noticed the helper. But did it also catch the whitelist
ordering issue? The trace found BOTH because it traces the complete
execution path, not just the failing line."

**/trace-execution focuses only on the exact-match issue:**
Follow up: "Now trace the scenario where cdn.example.com is whitelisted
and example.com is blocked. What happens with the current whitelist check
ordering?"
