import unittest

import beta_perf


class BetaPerfTest(unittest.TestCase):
    def test_normalize_counts_requires_strict_growth(self):
        self.assertEqual(beta_perf.normalize_counts("10,25,50"), [10, 25, 50])
        with self.assertRaises(ValueError):
            beta_perf.normalize_counts("10,10")
        with self.assertRaises(ValueError):
            beta_perf.normalize_counts("0")

    def test_fixture_payload_rotates_modes(self):
        repo = "https://github.com/example/repo"
        static = beta_perf.project_payload(0, "run", repo, "main")
        dockerfile = beta_perf.project_payload(1, "run", repo, "main")
        compose = beta_perf.project_payload(2, "run", repo, "main")
        self.assertEqual(static["deployMode"], "static")
        self.assertEqual(dockerfile["deployMode"], "dockerfile")
        self.assertEqual(compose["deployMode"], "compose")
        self.assertEqual(compose["mainService"], "app")
        self.assertEqual(compose["composeFilePath"], "compose.yml")

    def test_summary_detects_duplicate_ports(self):
        rows = [
            beta_perf.ProjectResult(index=1, fixture="dockerfile", name="a", allocated_port=3001, total_seconds=2.0, status="running"),
            beta_perf.ProjectResult(index=2, fixture="compose", name="b", allocated_port=3001, total_seconds=3.0, status="running"),
        ]
        summary = beta_perf.summarize_batch(rows, threshold_failure_rate=0.0, threshold_p95_seconds=10.0)
        self.assertFalse(summary["pass"])
        self.assertEqual(summary["duplicateAllocatedPorts"], [3001])

    def test_summary_enforces_failure_and_latency_thresholds(self):
        rows = [
            beta_perf.ProjectResult(index=1, fixture="static", name="a", total_seconds=1.0, status="running"),
            beta_perf.ProjectResult(index=2, fixture="dockerfile", name="b", total_seconds=20.0, status="failed"),
        ]
        summary = beta_perf.summarize_batch(rows, threshold_failure_rate=0.1, threshold_p95_seconds=5.0)
        self.assertFalse(summary["pass"])
        self.assertEqual(summary["failed"], 1)
        self.assertGreater(summary["failureRate"], 0.1)
        self.assertGreater(summary["p95TotalSeconds"], 5.0)

    def test_markdown_report_contains_batch_result(self):
        report = {
            "runId": "run-1",
            "gitSha": "abc123",
            "baseUrl": "https://mypaas.example",
            "startedAt": "2026-08-14T00:00:00Z",
            "finishedAt": "2026-08-14T00:01:00Z",
            "pass": True,
            "blockedReasons": [],
            "batches": [{
                "targetCount": 10,
                "summary": {
                    "running": 10,
                    "failed": 0,
                    "failureRate": 0.0,
                    "p95TotalSeconds": 4.2,
                    "pass": True,
                    "failures": [],
                },
            }],
        }
        text = beta_perf.markdown_report(report)
        self.assertIn("MyPaas many-project performance report", text)
        self.assertIn("| 10 | 10 | 0 | 0.000 | 4.20 | PASS |", text)


if __name__ == "__main__":
    unittest.main()
