# Phase 2 Full Backup + Fresh-VM Restore Summary

Gate state: `BLOCKED_ON_VM_EVIDENCE`

Blocked reason:

Phase 1 failed before the VM could be updated to the candidate release images. Fresh-VM restore evidence must be produced from the exact candidate runtime/source SHA.

No candidate runtime evidence was produced for this gate. The VM remains on `b94361be993963c0ad04a5c3a76130bc8d8ef8ee`; running this phase now would validate the old runtime rather than the requested candidate `edea8615d75f8e032bfb66bf430ab598b05876b2`.
