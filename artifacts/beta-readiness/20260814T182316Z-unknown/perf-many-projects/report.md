# MyPaas many-project performance report

- Run ID: `20260814T182316Z-unknown`
- Git SHA: `unknown`
- Target: `https://malala.tech`
- Fixture branch: `test/beta-fixture`
- Fixture resolved SHA: `91ab957ba072fcc96ffb7b78727d110c61826de4`
- Started: `2026-08-14T18:23:16Z`
- Finished: `2026-08-14T18:23:19Z`
- Overall: **FAIL**

| Target projects | Running | Failed | Failure rate | p95 total (s) | Result |
| ---: | ---: | ---: | ---: | ---: | --- |
| 10 | 0 | 10 | 1.000 | 0.09 | FAIL |
| 25 | 0 | 25 | 1.000 | 0.14 | FAIL |
| 50 | 0 | 50 | 1.000 | 0.10 | FAIL |

## Findings

- Batch 10: failure_rate 1.000 exceeds threshold 0.000
- Batch 25: failure_rate 1.000 exceeds threshold 0.000
- Batch 50: failure_rate 1.000 exceeds threshold 0.000

## Blocked observations

- host storage telemetry was unavailable from /api/admin/host-stats
- SSH Docker/cache attribution probe was not available
