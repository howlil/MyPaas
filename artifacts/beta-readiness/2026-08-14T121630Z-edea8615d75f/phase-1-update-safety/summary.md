# Phase 1 Update / Release Safety Summary

Gate state: `FAIL`

Results:

- Successful N to N+1 update: `FAIL`
- Missing-image safety: `PASS`
- Forced post-update verification rollback: `BLOCKED_ON_VM_EVIDENCE`

Root cause:

Candidate release images for `edea8615d75f8e032bfb66bf430ab598b05876b2` are not published to GHCR under the immutable full commit SHA tags required by the updater. The updater correctly refused to advance and preserved the running `b94361be993963c0ad04a5c3a76130bc8d8ef8ee` runtime.
