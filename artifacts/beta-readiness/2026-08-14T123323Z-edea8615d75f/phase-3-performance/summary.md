# Phase 3 10 / 25 / 50 Project Performance Summary

Gate state: `FAIL`

`benchmarks/beta_perf.py` was run with `--fixture-ref edea8615d75f8e032bfb66bf430ab598b05876b2` and `--git-sha edea8615d75f8e032bfb66bf430ab598b05876b2`, as required.

The 10-project stage failed at project creation with 100% failure rate. The API rejected the exact SHA because the harness sends `fixture_ref` as the project `branch`:

```text
validation failed: failed to clone repository branch "edea8615d75f8e032bfb66bf430ab598b05876b2"
```

The 25/50 stages cannot produce valid capacity evidence because the initial stage could not create any projects.

Root cause:

Harness/API contract defect: the beta harness cannot evaluate an exact commit SHA fixture ref against an API contract that only accepts cloneable branch names for project creation.
