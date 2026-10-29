# Status digest

Push it when something changes; stay silent otherwise. Lead with the headline metric.

```
<headline metric>: <x> → <y> (Δ<z>%)   p50 <a> / p90 <b>
Merged: <n> (+<k> since last)   Open: <n>   Blocked: <n>
Last merge: <t> ago                     ← watchdog: alert past <threshold>
ETA: <time, operator's timezone>  (previous forecast: <time>; delta: <±>)

| Milestone | State | Evidence |
|---|---|---|
| <name> | claimed / pushed / green / merged / deployed / measured | <link> |

Needs you: <one question, or "nothing">
```

Evidence levels, weakest to strongest: claimed < pushed < green < merged < deployed < measured. Never report a level higher than the evidence supports.
