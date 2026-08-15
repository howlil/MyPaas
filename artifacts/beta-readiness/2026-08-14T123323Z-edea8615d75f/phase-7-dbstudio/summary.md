# Phase 7 DB Studio Compose Reliability Summary

Gate state: `FAIL`

`benchmarks/dbstudio_compose_smoke.py` was run with `--fixture-ref edea8615d75f8e032bfb66bf430ab598b05876b2`.

PostgreSQL, MySQL, and MariaDB fixture project creation all failed before deployment because the exact SHA was sent as the project branch and rejected by the API:

```text
validation failed: failed to clone repository branch "edea8615d75f8e032bfb66bf430ab598b05876b2"
```

Root cause:

Same harness/API contract defect as Phase 3 and Phase 4: exact SHA fixture refs are treated as branch names.
