#!/usr/bin/env python3
"""Controlled many-project beta-readiness harness for MyPaas.

The harness is destructive: it creates and deploys projects. Use --plan to inspect
the workload without sending requests. Real runs require --confirm-destructive.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import datetime as dt
import json
import os
import pathlib
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass
from typing import Any

TERMINAL_DEPLOYMENT_STATES = {"running", "failed", "stopped", "rolled_back"}
FIXTURES = (
    {
        "kind": "static",
        "deployMode": "static",
        "resourceProfile": "static",
        "appPort": 80,
        "baseDirectory": "benchmarks/fixtures/beta/static",
        "staticFrontendPath": ".",
    },
    {
        "kind": "dockerfile",
        "deployMode": "dockerfile",
        "resourceProfile": "node-python",
        "appPort": 8080,
        "baseDirectory": "benchmarks/fixtures/beta/dockerfile",
    },
    {
        "kind": "compose",
        "deployMode": "compose",
        "resourceProfile": "compose-main",
        "appPort": 8080,
        "baseDirectory": "benchmarks/fixtures/beta/compose",
        "mainService": "app",
        "composeFilePath": "compose.yml",
    },
)


class HarnessError(RuntimeError):
    pass


@dataclass
class ProjectResult:
    index: int
    fixture: str
    name: str
    project_id: str | None = None
    deployment_id: str | None = None
    allocated_port: int | None = None
    create_seconds: float | None = None
    deploy_seconds: float | None = None
    total_seconds: float | None = None
    status: str = "not_started"
    error: str | None = None


class ApiClient:
    def __init__(self, base_url: str, token: str, timeout: float = 60.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.timeout = timeout

    def request(self, method: str, path: str, payload: dict[str, Any] | None = None) -> Any:
        body = None
        headers = {"Accept": "application/json", "Authorization": f"Bearer {self.token}"}
        if payload is not None:
            body = json.dumps(payload).encode("utf-8")
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(
            f"{self.base_url}{path}",
            data=body,
            headers=headers,
            method=method,
        )
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise HarnessError(f"{method} {path}: HTTP {exc.code}: {redact(detail)}") from exc
        except urllib.error.URLError as exc:
            raise HarnessError(f"{method} {path}: {exc.reason}") from exc

        if not raw:
            return None
        decoded = json.loads(raw.decode("utf-8"))
        if isinstance(decoded, dict) and "data" in decoded:
            return decoded["data"]
        return decoded


def redact(value: str) -> str:
    text = value
    for marker in ("Bearer ", "token=", "password=", "secret="):
        pos = text.lower().find(marker.lower())
        if pos >= 0:
            end = text.find(" ", pos + len(marker))
            if end < 0:
                end = len(text)
            text = text[: pos + len(marker)] + "<redacted>" + text[end:]
    return text[:2000]


def normalize_counts(raw: str) -> list[int]:
    counts: list[int] = []
    for item in raw.split(","):
        item = item.strip()
        if not item:
            continue
        value = int(item)
        if value <= 0:
            raise ValueError("project counts must be positive")
        if counts and value <= counts[-1]:
            raise ValueError("project counts must be strictly increasing")
        counts.append(value)
    if not counts:
        raise ValueError("at least one project count is required")
    return counts


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def safe_sha(value: str) -> str:
    value = "".join(ch for ch in value if ch.isalnum())
    return value[:12] or "unknown"


def project_payload(index: int, prefix: str, fixture_repo: str, fixture_ref: str) -> dict[str, Any]:
    fixture = FIXTURES[index % len(FIXTURES)]
    suffix = f"{index + 1:03d}-{fixture['kind']}"
    payload: dict[str, Any] = {
        "name": f"{prefix}-{suffix}",
        "sourceType": "git",
        "repoUrl": fixture_repo,
        "branch": fixture_ref,
        "deployMode": fixture["deployMode"],
        "resourceProfile": fixture["resourceProfile"],
        "appPort": fixture["appPort"],
        "baseDirectory": fixture["baseDirectory"],
    }
    for key in ("staticFrontendPath", "mainService", "composeFilePath"):
        if key in fixture:
            payload[key] = fixture[key]
    return payload


def wait_for_deployment(client: ApiClient, deployment_id: str, timeout_seconds: float, poll_seconds: float) -> dict[str, Any]:
    deadline = time.monotonic() + timeout_seconds
    last: dict[str, Any] = {}
    while time.monotonic() < deadline:
        current = client.request("GET", f"/api/deployments/{deployment_id}")
        if isinstance(current, dict):
            last = current
            if current.get("status") in TERMINAL_DEPLOYMENT_STATES:
                return current
        time.sleep(poll_seconds)
    raise HarnessError(
        f"deployment {deployment_id} did not reach a terminal state within {timeout_seconds:.0f}s "
        f"(last={last.get('status', 'unknown')})"
    )


def run_one(
    client: ApiClient,
    index: int,
    prefix: str,
    fixture_repo: str,
    fixture_ref: str,
    deploy_timeout: float,
    poll_seconds: float,
) -> ProjectResult:
    payload = project_payload(index, prefix, fixture_repo, fixture_ref)
    result = ProjectResult(index=index + 1, fixture=str(payload["deployMode"]), name=str(payload["name"]))
    started = time.monotonic()
    try:
        create_started = time.monotonic()
        project = client.request("POST", "/api/projects/", payload)
        result.create_seconds = time.monotonic() - create_started
        if not isinstance(project, dict) or not project.get("id"):
            raise HarnessError("create project response did not include an id")
        result.project_id = str(project["id"])
        port = project.get("allocatedPort")
        result.allocated_port = int(port) if port is not None else None

        deploy_started = time.monotonic()
        deployment = client.request("POST", f"/api/projects/{result.project_id}/deploy")
        if not isinstance(deployment, dict) or not deployment.get("id"):
            raise HarnessError("trigger deployment response did not include an id")
        result.deployment_id = str(deployment["id"])
        final = wait_for_deployment(client, result.deployment_id, deploy_timeout, poll_seconds)
        result.deploy_seconds = time.monotonic() - deploy_started
        result.status = str(final.get("status", "unknown"))
        if result.status != "running":
            result.error = redact(str(final.get("errorMsg") or f"terminal status {result.status}"))
    except Exception as exc:
        result.status = "harness_error"
        result.error = redact(str(exc))
    finally:
        result.total_seconds = time.monotonic() - started
    return result


def host_api_snapshot(client: ApiClient) -> dict[str, Any]:
    try:
        value = client.request("GET", "/api/admin/host-stats")
        return value if isinstance(value, dict) else {"available": False}
    except Exception as exc:
        return {"available": False, "error": redact(str(exc))}


def ssh_host_snapshot(target: str | None) -> dict[str, Any]:
    if not target:
        return {"available": False, "reason": "ssh target not configured"}
    command = [
        "ssh",
        "-o",
        "BatchMode=yes",
        target,
        (
            "set -eu; "
            "echo '=== docker-system-df ==='; docker system df; "
            "echo '=== buildkit-cache ==='; docker builder du 2>/dev/null || true; "
            "echo '=== mypaas-artifacts ==='; "
            "du -sb /var/lib/mypaas/static /var/lib/mypaas/backups /var/lib/mypaas/compose "
            "/var/lib/mypaas/volumes 2>/dev/null || true"
        ),
    ]
    try:
        completed = subprocess.run(command, capture_output=True, text=True, timeout=30, check=False)
    except (OSError, subprocess.SubprocessError) as exc:
        return {"available": False, "error": redact(str(exc))}
    return {
        "available": completed.returncode == 0,
        "returnCode": completed.returncode,
        "stdout": redact(completed.stdout),
        "stderr": redact(completed.stderr),
    }


def duplicate_ports(results: list[ProjectResult]) -> list[int]:
    seen: set[int] = set()
    duplicates: set[int] = set()
    for item in results:
        if item.allocated_port is None:
            continue
        if item.allocated_port in seen:
            duplicates.add(item.allocated_port)
        seen.add(item.allocated_port)
    return sorted(duplicates)


def percentile(values: list[float], fraction: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = max(0, min(len(ordered) - 1, int(round((len(ordered) - 1) * fraction))))
    return ordered[index]


def summarize_batch(results: list[ProjectResult], threshold_failure_rate: float, threshold_p95_seconds: float) -> dict[str, Any]:
    successes = [item for item in results if item.status == "running"]
    timings = [float(item.total_seconds) for item in results if item.total_seconds is not None]
    failure_rate = (len(results) - len(successes)) / len(results) if results else 1.0
    p95 = percentile(timings, 0.95)
    duplicates = duplicate_ports(results)
    failures: list[str] = []
    if failure_rate > threshold_failure_rate:
        failures.append(f"failure_rate {failure_rate:.3f} exceeds threshold {threshold_failure_rate:.3f}")
    if p95 is not None and p95 > threshold_p95_seconds:
        failures.append(f"p95_total_seconds {p95:.2f} exceeds threshold {threshold_p95_seconds:.2f}")
    if duplicates:
        failures.append(f"duplicate allocated ports: {duplicates}")
    return {
        "projects": len(results),
        "running": len(successes),
        "failed": len(results) - len(successes),
        "failureRate": failure_rate,
        "p50TotalSeconds": percentile(timings, 0.50),
        "p95TotalSeconds": p95,
        "maxTotalSeconds": max(timings) if timings else None,
        "duplicateAllocatedPorts": duplicates,
        "pass": not failures,
        "failures": failures,
    }


def write_report(report: dict[str, Any], output_dir: pathlib.Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    (output_dir / "report.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    (output_dir / "report.md").write_text(markdown_report(report), encoding="utf-8")


def markdown_report(report: dict[str, Any]) -> str:
    lines = [
        "# MyPaas many-project performance report",
        "",
        f"- Run ID: `{report['runId']}`",
        f"- Git SHA: `{report['gitSha']}`",
        f"- Target: `{report['baseUrl']}`",
        f"- Started: `{report['startedAt']}`",
        f"- Finished: `{report['finishedAt']}`",
        f"- Overall: **{'PASS' if report['pass'] else 'FAIL'}**",
        "",
        "| Target projects | Running | Failed | Failure rate | p95 total (s) | Result |",
        "| ---: | ---: | ---: | ---: | ---: | --- |",
    ]
    for batch in report["batches"]:
        summary = batch["summary"]
        p95 = summary["p95TotalSeconds"]
        p95_text = f"{p95:.2f}" if p95 is not None else "n/a"
        lines.append(
            f"| {batch['targetCount']} | {summary['running']} | {summary['failed']} | "
            f"{summary['failureRate']:.3f} | {p95_text} | {'PASS' if summary['pass'] else 'FAIL'} |"
        )
    lines.extend(["", "## Findings", ""])
    findings: list[str] = []
    for batch in report["batches"]:
        for failure in batch["summary"]["failures"]:
            findings.append(f"- Batch {batch['targetCount']}: {failure}")
    if not findings:
        findings.append("- No configured threshold or port-consistency failures.")
    lines.extend(findings)
    if report.get("blockedReasons"):
        lines.extend(["", "## Blocked observations", ""])
        lines.extend(f"- {item}" for item in report["blockedReasons"])
    lines.append("")
    return "\n".join(lines)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default=os.getenv("MYPAAS_BETA_BASE_URL", ""))
    parser.add_argument("--token", default=os.getenv("MYPAAS_API_TOKEN", ""))
    parser.add_argument("--fixture-repo", default=os.getenv("MYPAAS_BETA_FIXTURE_REPO", "https://github.com/nabilrn/MyPaas"))
    parser.add_argument("--fixture-ref", default=os.getenv("MYPAAS_BETA_FIXTURE_REF", "main"))
    parser.add_argument("--git-sha", default=os.getenv("MYPAAS_BETA_GIT_SHA", "unknown"))
    parser.add_argument("--counts", default="10,25,50")
    parser.add_argument("--parallel", type=int, default=2)
    parser.add_argument("--deploy-timeout", type=float, default=900)
    parser.add_argument("--poll-seconds", type=float, default=3)
    parser.add_argument("--max-failure-rate", type=float, default=0.0)
    parser.add_argument("--max-p95-seconds", type=float, default=900.0)
    parser.add_argument("--ssh-target", default=os.getenv("MYPAAS_BETA_SSH_TARGET", ""))
    parser.add_argument("--output", default="")
    parser.add_argument("--prefix", default="")
    parser.add_argument("--plan", action="store_true")
    parser.add_argument("--confirm-destructive", action="store_true")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    counts = normalize_counts(args.counts)
    if args.parallel < 1:
        raise SystemExit("--parallel must be at least 1")
    if not 0 <= args.max_failure_rate <= 1:
        raise SystemExit("--max-failure-rate must be between 0 and 1")
    if args.plan:
        print(json.dumps({
            "counts": counts,
            "fixtures": list(FIXTURES),
            "fixtureRepo": args.fixture_repo,
            "fixtureRef": args.fixture_ref,
            "parallel": args.parallel,
            "destructive": True,
        }, indent=2))
        return 0
    if not args.confirm_destructive:
        raise SystemExit("refusing to create projects without --confirm-destructive; use --plan first")
    if not args.base_url:
        raise SystemExit("--base-url or MYPAAS_BETA_BASE_URL is required")
    if not args.token:
        raise SystemExit("--token or MYPAAS_API_TOKEN is required")

    started_at = utc_now()
    stamp = started_at.replace(":", "").replace("-", "")
    prefix = args.prefix or f"beta-perf-{stamp.lower().replace('t', '-')[:18]}"
    run_id = f"{stamp}-{safe_sha(args.git_sha)}"
    output_dir = pathlib.Path(args.output or f"artifacts/beta-readiness/{run_id}/perf-many-projects")
    client = ApiClient(args.base_url, args.token)
    all_results: list[ProjectResult] = []
    batches: list[dict[str, Any]] = []
    blocked: list[str] = []

    before_api = host_api_snapshot(client)
    before_ssh = ssh_host_snapshot(args.ssh_target or None)

    for target_count in counts:
        needed = target_count - len(all_results)
        indices = list(range(len(all_results), len(all_results) + needed))
        stage_started = time.monotonic()
        stage_results: list[ProjectResult] = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=args.parallel) as executor:
            futures = [
                executor.submit(
                    run_one,
                    client,
                    index,
                    prefix,
                    args.fixture_repo,
                    args.fixture_ref,
                    args.deploy_timeout,
                    args.poll_seconds,
                )
                for index in indices
            ]
            for future in concurrent.futures.as_completed(futures):
                stage_results.append(future.result())
        stage_results.sort(key=lambda item: item.index)
        all_results.extend(stage_results)
        summary = summarize_batch(all_results, args.max_failure_rate, args.max_p95_seconds)
        batches.append({
            "targetCount": target_count,
            "stageSeconds": time.monotonic() - stage_started,
            "summary": summary,
            "projects": [asdict(item) for item in all_results],
        })

    after_api = host_api_snapshot(client)
    after_ssh = ssh_host_snapshot(args.ssh_target or None)
    if not before_api.get("storage") or not after_api.get("storage"):
        blocked.append("host storage telemetry was unavailable from /api/admin/host-stats")
    if not before_ssh.get("available") or not after_ssh.get("available"):
        blocked.append("SSH Docker/cache attribution probe was not available")

    report = {
        "schemaVersion": 1,
        "kind": "perf-many-projects",
        "runId": run_id,
        "gitSha": args.git_sha,
        "baseUrl": args.base_url,
        "fixtureRepo": args.fixture_repo,
        "fixtureRef": args.fixture_ref,
        "startedAt": started_at,
        "finishedAt": utc_now(),
        "thresholds": {"maxFailureRate": args.max_failure_rate, "maxP95Seconds": args.max_p95_seconds},
        "hostBefore": {"api": before_api, "ssh": before_ssh},
        "hostAfter": {"api": after_api, "ssh": after_ssh},
        "batches": batches,
        "blockedReasons": blocked,
        "pass": bool(batches) and all(batch["summary"]["pass"] for batch in batches),
    }
    write_report(report, output_dir)
    print(output_dir)
    return 0 if report["pass"] else 1


if __name__ == "__main__":
    sys.exit(main())
