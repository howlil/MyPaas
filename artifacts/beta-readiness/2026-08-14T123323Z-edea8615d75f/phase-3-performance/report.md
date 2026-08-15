# MyPaas many-project performance report

- Run ID: `20260814T124007Z-edea8615d75f`
- Git SHA: `edea8615d75f8e032bfb66bf430ab598b05876b2`
- Target: `http://127.0.0.1:8080`
- Started: `2026-08-14T12:40:07Z`
- Finished: `2026-08-14T12:40:22Z`
- Overall: **FAIL**

| Target projects | Running | Failed | Failure rate | p95 total (s) | Result |
| ---: | ---: | ---: | ---: | ---: | --- |
| 10 | 0 | 10 | 1.000 | 0.64 | FAIL |
| 25 | 0 | 25 | 1.000 | 0.63 | FAIL |
| 50 | 0 | 50 | 1.000 | 0.63 | FAIL |

## Findings

- Batch 10: failure_rate 1.000 exceeds threshold 0.000
- Batch 25: failure_rate 1.000 exceeds threshold 0.000
- Batch 50: failure_rate 1.000 exceeds threshold 0.000

## Blocked observations

- SSH Docker/cache attribution probe was not available
