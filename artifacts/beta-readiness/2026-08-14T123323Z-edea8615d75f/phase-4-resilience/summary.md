# Phase 4 Concurrent Deployment Resilience Summary

Gate state: `FAIL`

`benchmarks/beta_resilience.py` was run with `--fixture-ref edea8615d75f8e032bfb66bf430ab598b05876b2`.

It failed before the concurrent scenario could start because project creation rejected the exact SHA as a branch:

```text
validation failed: failed to clone repository branch "edea8615d75f8e032bfb66bf430ab598b05876b2"
```

Root cause:

Same harness/API contract defect as Phase 3: exact SHA fixture refs are treated as branch names.
