# DB Studio Compose smoke report

- Run ID: `20260814T124254Z-edea8615d75f`
- Git SHA: `edea8615d75f8e032bfb66bf430ab598b05876b2`
- Result: **FAIL**

| Engine | Deployment | Configured | Connected | Driver | Read-only default | Schemas | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| postgres | not_started | False | False | n/a | False | False | FAIL |
| mysql | not_started | False | False | n/a | False | False | FAIL |
| mariadb | not_started | False | False | n/a | False | False | FAIL |

## Failures

- postgres: POST /api/projects/: HTTP 400: {"error":{"code":"VALIDATION_FAILED","message":"validation failed: failed to clone repository branch \"edea8615d75f8e032bfb66bf430ab598b05876b2\""}}

- mysql: POST /api/projects/: HTTP 400: {"error":{"code":"VALIDATION_FAILED","message":"validation failed: failed to clone repository branch \"edea8615d75f8e032bfb66bf430ab598b05876b2\""}}

- mariadb: POST /api/projects/: HTTP 400: {"error":{"code":"VALIDATION_FAILED","message":"validation failed: failed to clone repository branch \"edea8615d75f8e032bfb66bf430ab598b05876b2\""}}

