# Phase 1 Update / Release Safety Summary

Gate state: `FAIL`

Results:

- Successful N to candidate update with exact full-SHA images: `PASS`
- Missing-image safety after candidate update: `PASS`
- Forced post-update verification rollback from the VM's initial `b94361be993963c0ad04a5c3a76130bc8d8ef8ee` updater: `FAIL`
- Candidate updater local rollback image path: `PARTIAL`/diagnostic; it used local `rollback-edea8615d75f` images, and explicit verification passed after containers settled.

Root cause:

The VM initially ran `b94361be993963c0ad04a5c3a76130bc8d8ef8ee`. Its already-running `scripts/update-vm.sh` attempted to pull `rollback-b94361be9939` from GHCR during rollback instead of using verified local rollback images. This is the exact rollback-path defect the handoff called out. The candidate source contains the corrected local-image rollback path, but that fix is not available to the already-running old updater during a failed upgrade from `b94361be993963c0ad04a5c3a76130bc8d8ef8ee`.

Evidence:

- `rollback-drill-command.txt`
- `rollback-before-runtime.txt`
- `rollback-after-runtime.txt`
- `update-success-command.txt`
- `update-success-verify-production.txt`
- `missing-image-command.txt`
- `missing-image-verify-production.txt`
- `candidate-script-rollback-command.txt`
- `candidate-rollback-explicit-verify.txt`
- `restore-exact-candidate-tags.txt`
