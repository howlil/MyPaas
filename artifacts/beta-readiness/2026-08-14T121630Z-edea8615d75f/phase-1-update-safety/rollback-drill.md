# Forced Post-update Verification Failure / Rollback Drill

Status: `BLOCKED_ON_VM_EVIDENCE`

The rollback drill requires a target update to proceed past image selection and deployment. Candidate release images are unavailable, so the updater never advances to the point where a forced post-update verification failure can be injected.
