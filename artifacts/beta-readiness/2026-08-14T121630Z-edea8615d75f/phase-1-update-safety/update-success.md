# Successful N to N+1 Update

Status: `FAIL`

The required candidate release images are not published:

- `ghcr.io/nabilrn/mypaas-api:edea8615d75f8e032bfb66bf430ab598b05876b2`: unavailable
- `ghcr.io/nabilrn/mypaas-dashboard:edea8615d75f8e032bfb66bf430ab598b05876b2`: unavailable

`scripts/update-vm.sh` requires registry-published immutable full-SHA release image tags before it advances the checkout. Because the images are missing, the required successful N to N+1 candidate update could not be executed.
