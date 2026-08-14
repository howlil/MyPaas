#!/usr/bin/env python3
"""Full MyPaas backup/restore bundle tooling for controlled beta drills.

The existing in-process backup scheduler remains the lightweight control-plane
PostgreSQL backup. This tool creates a fuller disaster-recovery bundle that also
captures production configuration, static artifacts, Compose workspaces, and
MyPaas-managed project volumes.

Backup bundles contain secrets in private/config.env and therefore must be
handled as sensitive operational artifacts. Reports and manifests never include
configuration values.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tarfile
import tempfile
import time
from dataclasses import dataclass
from typing import Any, Iterable

SCHEMA_VERSION = 1
DEFAULT_INSTALL_DIR = "/opt/mypaas"
DEFAULT_BACKUP_ROOT = "/var/lib/mypaas/backups"
DEFAULT_STATIC_ROOT = "/var/lib/mypaas/static"
DEFAULT_COMPOSE_ROOT = "/var/lib/mypaas/compose"
MANAGED_LABEL = "mypaas.managed"
MANAGED_VALUE = "true"


class BackupRestoreError(RuntimeError):
    pass


@dataclass(frozen=True)
class ManagedVolume:
    name: str
    mountpoint: str
    source: str


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def parse_env_file(path: pathlib.Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
            value = value[1:-1]
        values[key] = value
    return values


def required_env(values: dict[str, str], key: str) -> str:
    value = values.get(key, "").strip()
    if not value:
        raise BackupRestoreError(f"required production setting {key} is missing")
    return value


def run(
    args: list[str],
    *,
    input_bytes: bytes | None = None,
    stdout_file: pathlib.Path | None = None,
    check: bool = True,
) -> subprocess.CompletedProcess[bytes]:
    stdout: Any = subprocess.PIPE
    handle = None
    if stdout_file is not None:
        stdout_file.parent.mkdir(parents=True, exist_ok=True)
        handle = stdout_file.open("wb")
        stdout = handle
    try:
        completed = subprocess.run(
            args,
            input=input_bytes,
            stdout=stdout,
            stderr=subprocess.PIPE,
            check=False,
        )
    except OSError as exc:
        raise BackupRestoreError(f"cannot execute {args[0]}: {exc}") from exc
    finally:
        if handle is not None:
            handle.close()
    if check and completed.returncode != 0:
        message = completed.stderr.decode("utf-8", errors="replace").strip()
        raise BackupRestoreError(f"command failed ({args[0]}): {message[:1000]}")
    return completed


def sha256_file(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def file_record(kind: str, bundle: pathlib.Path, path: pathlib.Path, **extra: Any) -> dict[str, Any]:
    record = {
        "kind": kind,
        "path": path.relative_to(bundle).as_posix(),
        "sha256": sha256_file(path),
        "sizeBytes": path.stat().st_size,
    }
    record.update(extra)
    return record


def archive_directory(source: pathlib.Path, destination: pathlib.Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(destination, "w:gz", format=tarfile.PAX_FORMAT) as archive:
        if source.exists():
            for child in sorted(source.iterdir(), key=lambda p: p.name):
                archive.add(child, arcname=child.name, recursive=True)
    os.chmod(destination, 0o600)


def _safe_member_destination(root: pathlib.Path, name: str) -> pathlib.Path:
    candidate = (root / name).resolve()
    resolved_root = root.resolve()
    try:
        candidate.relative_to(resolved_root)
    except ValueError as exc:
        raise BackupRestoreError(f"unsafe archive member path: {name}") from exc
    return candidate


def safe_extract(archive_path: pathlib.Path, destination: pathlib.Path) -> None:
    destination.mkdir(parents=True, exist_ok=True)
    with tarfile.open(archive_path, "r:gz") as archive:
        for member in archive.getmembers():
            _safe_member_destination(destination, member.name)
            if member.issym() or member.islnk():
                link_target = pathlib.PurePosixPath(member.name).parent / member.linkname
                if link_target.is_absolute() or ".." in link_target.parts:
                    raise BackupRestoreError(f"unsafe archive link: {member.name}")
        archive.extractall(destination)


def clear_directory(path: pathlib.Path) -> None:
    path.mkdir(parents=True, exist_ok=True)
    for child in path.iterdir():
        if child.is_symlink() or child.is_file():
            child.unlink()
        else:
            shutil.rmtree(child)


def docker_json(args: list[str]) -> Any:
    completed = run(["docker", *args])
    raw = completed.stdout.decode("utf-8", errors="replace").strip()
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        raise BackupRestoreError(f"docker returned invalid JSON for {' '.join(args)}") from exc


def active_project_names(postgres_user: str, postgres_db: str) -> set[str]:
    query = "SELECT name FROM projects WHERE deleted_at IS NULL ORDER BY name"
    completed = run([
        "docker",
        "exec",
        "mypaas-postgres-prod",
        "psql",
        "-X",
        "-A",
        "-t",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
        "-c",
        query,
    ])
    return {line.strip() for line in completed.stdout.decode().splitlines() if line.strip()}


def classify_volume(info: dict[str, Any], project_names: set[str]) -> str | None:
    labels = info.get("Labels") or {}
    name = str(info.get("Name") or "")
    if str(labels.get(MANAGED_LABEL, "")).lower() == MANAGED_VALUE:
        return "mypaas-managed"
    compose_project = str(labels.get("com.docker.compose.project") or "")
    if compose_project and compose_project in project_names:
        return "mypaas-compose"
    # Older Compose-created volumes can survive label changes. The exact active
    # project-name prefix is still constrained by the control-plane project list.
    if any(name.startswith(project + "_") for project in project_names):
        return "mypaas-compose-legacy"
    return None


def discover_managed_volumes(project_names: set[str]) -> list[ManagedVolume]:
    completed = run(["docker", "volume", "ls", "-q"])
    names = sorted({line.strip() for line in completed.stdout.decode().splitlines() if line.strip()})
    result: list[ManagedVolume] = []
    for name in names:
        rows = docker_json(["volume", "inspect", name])
        if not isinstance(rows, list) or not rows:
            continue
        info = rows[0]
        source = classify_volume(info, project_names)
        if source is None:
            continue
        mountpoint = str(info.get("Mountpoint") or "").strip()
        if not mountpoint:
            raise BackupRestoreError(f"managed volume {name} has no mountpoint")
        result.append(ManagedVolume(name=name, mountpoint=mountpoint, source=source))
    return result


def running_containers_for_volume(name: str) -> list[str]:
    completed = run(["docker", "ps", "-q", "--filter", f"volume={name}"])
    return [line.strip() for line in completed.stdout.decode().splitlines() if line.strip()]


def stop_containers(ids: Iterable[str]) -> list[str]:
    unique = sorted(set(ids))
    if unique:
        run(["docker", "stop", *unique])
    return unique


def start_containers(ids: Iterable[str]) -> None:
    unique = sorted(set(ids))
    if unique:
        run(["docker", "start", *unique])


def git_sha(install_dir: pathlib.Path) -> str:
    completed = run(["git", "-c", f"safe.directory={install_dir}", "-C", str(install_dir), "rev-parse", "HEAD"])
    return completed.stdout.decode().strip()


def dump_control_database(output: pathlib.Path, postgres_user: str, postgres_db: str) -> None:
    run([
        "docker",
        "exec",
        "mypaas-postgres-prod",
        "pg_dump",
        "--format=custom",
        "--no-owner",
        "--no-privileges",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
    ], stdout_file=output)
    os.chmod(output, 0o600)


def restore_control_database(dump_path: pathlib.Path, postgres_user: str, postgres_db: str) -> None:
    payload = dump_path.read_bytes()
    run([
        "docker",
        "exec",
        "-i",
        "mypaas-postgres-prod",
        "pg_restore",
        "--clean",
        "--if-exists",
        "--no-owner",
        "--no-privileges",
        "-U",
        postgres_user,
        "-d",
        postgres_db,
    ], input_bytes=payload)


def ensure_postgres(install_dir: pathlib.Path, env_file: pathlib.Path, compose_file: pathlib.Path, postgres_user: str, postgres_db: str) -> None:
    run([
        "docker",
        "compose",
        "-f",
        str(compose_file),
        "--env-file",
        str(env_file),
        "up",
        "-d",
        "postgres",
    ])
    deadline = time.monotonic() + 90
    while time.monotonic() < deadline:
        completed = run([
            "docker",
            "compose",
            "-f",
            str(compose_file),
            "--env-file",
            str(env_file),
            "exec",
            "-T",
            "postgres",
            "pg_isready",
            "-U",
            postgres_user,
            "-d",
            postgres_db,
        ], check=False)
        if completed.returncode == 0:
            return
        time.sleep(2)
    raise BackupRestoreError("restored PostgreSQL did not become ready within 90 seconds")


def manifest_path(bundle: pathlib.Path) -> pathlib.Path:
    return bundle / "manifest.json"


def write_json_private(path: pathlib.Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.chmod(path, 0o600)


def load_manifest(bundle: pathlib.Path) -> dict[str, Any]:
    path = manifest_path(bundle)
    if not path.is_file():
        raise BackupRestoreError(f"missing backup manifest: {path}")
    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("schemaVersion") != SCHEMA_VERSION:
        raise BackupRestoreError(f"unsupported manifest schemaVersion {data.get('schemaVersion')}")
    if not isinstance(data.get("files"), list):
        raise BackupRestoreError("manifest files must be an array")
    return data


def verify_bundle(bundle: pathlib.Path) -> dict[str, Any]:
    manifest = load_manifest(bundle)
    checked = 0
    for record in manifest["files"]:
        rel = pathlib.PurePosixPath(str(record.get("path") or ""))
        if rel.is_absolute() or ".." in rel.parts:
            raise BackupRestoreError(f"unsafe manifest path: {rel}")
        path = bundle / pathlib.Path(*rel.parts)
        if not path.is_file():
            raise BackupRestoreError(f"missing backup file: {rel}")
        expected = str(record.get("sha256") or "")
        actual = sha256_file(path)
        if not expected or actual != expected:
            raise BackupRestoreError(f"checksum mismatch: {rel}")
        checked += 1
    return {
        "schemaVersion": 1,
        "kind": "backup-verify",
        "verifiedAt": utc_now(),
        "sourceGitSha": manifest.get("sourceGitSha", "unknown"),
        "filesChecked": checked,
        "status": "PASS",
    }


def backup(args: argparse.Namespace) -> int:
    install_dir = pathlib.Path(args.install_dir).resolve()
    env_file = pathlib.Path(args.env_file or install_dir / ".env").resolve()
    if not env_file.is_file():
        raise BackupRestoreError(f"missing production config: {env_file}")
    env = parse_env_file(env_file)
    postgres_user = required_env(env, "POSTGRES_USER")
    postgres_db = required_env(env, "POSTGRES_DB")

    source_sha = git_sha(install_dir)
    stamp = utc_now().replace(":", "").replace("-", "")
    output = pathlib.Path(args.output or f"{DEFAULT_BACKUP_ROOT}/full-{stamp}-{source_sha[:12]}").resolve()
    if output.exists() and any(output.iterdir()):
        raise BackupRestoreError(f"backup output already exists and is not empty: {output}")
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(output, 0o700)

    files: list[dict[str, Any]] = []
    private_config = output / "private" / "config.env"
    private_config.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    shutil.copyfile(env_file, private_config)
    os.chmod(private_config, 0o600)
    files.append(file_record("production-config", output, private_config))

    database_dump = output / "control-plane" / "postgres.dump"
    dump_control_database(database_dump, postgres_user, postgres_db)
    files.append(file_record("control-plane-postgres", output, database_dump))

    static_archive = output / "filesystem" / "static.tar.gz"
    archive_directory(pathlib.Path(args.static_root), static_archive)
    files.append(file_record("static-artifacts", output, static_archive))

    compose_archive = output / "filesystem" / "compose.tar.gz"
    archive_directory(pathlib.Path(args.compose_root), compose_archive)
    files.append(file_record("compose-workspaces", output, compose_archive))

    projects = active_project_names(postgres_user, postgres_db)
    volumes = discover_managed_volumes(projects)
    active: dict[str, list[str]] = {volume.name: running_containers_for_volume(volume.name) for volume in volumes}
    in_use = sorted({container for rows in active.values() for container in rows})
    if in_use and not args.quiesce_managed_containers:
        raise BackupRestoreError(
            "managed project volumes are mounted by running containers; re-run with "
            "--quiesce-managed-containers for a controlled consistent snapshot"
        )

    stopped: list[str] = []
    try:
        if in_use:
            stopped = stop_containers(in_use)
        for volume in volumes:
            archive = output / "volumes" / f"{volume.name}.tar.gz"
            archive_directory(pathlib.Path(volume.mountpoint), archive)
            files.append(file_record(
                "managed-volume",
                output,
                archive,
                volumeName=volume.name,
                discovery=volume.source,
            ))
    finally:
        if stopped:
            start_containers(stopped)

    manifest = {
        "schemaVersion": SCHEMA_VERSION,
        "kind": "mypaas-full-backup",
        "createdAt": utc_now(),
        "sourceGitSha": source_sha,
        "configurationIncluded": True,
        "configurationValuesExposedInManifest": False,
        "quiescedManagedContainers": bool(stopped),
        "projectCount": len(projects),
        "managedVolumeCount": len(volumes),
        "files": files,
    }
    write_json_private(manifest_path(output), manifest)
    report = {
        "schemaVersion": 1,
        "kind": "backup-create",
        "createdAt": manifest["createdAt"],
        "sourceGitSha": source_sha,
        "bundle": str(output),
        "projectCount": len(projects),
        "managedVolumeCount": len(volumes),
        "files": len(files),
        "status": "PASS",
        "containsSensitiveConfiguration": True,
    }
    write_json_private(output / "backup-report.json", report)
    print(output)
    return 0


def restore_volume(bundle: pathlib.Path, record: dict[str, Any]) -> None:
    name = str(record.get("volumeName") or "").strip()
    if not name:
        raise BackupRestoreError("managed-volume record is missing volumeName")
    archive = bundle / str(record["path"])
    running = running_containers_for_volume(name)
    if running:
        raise BackupRestoreError(f"refusing to restore in-use volume {name}; running containers: {len(running)}")
    run(["docker", "volume", "create", "--label", f"{MANAGED_LABEL}={MANAGED_VALUE}", name])
    rows = docker_json(["volume", "inspect", name])
    if not isinstance(rows, list) or not rows or not rows[0].get("Mountpoint"):
        raise BackupRestoreError(f"cannot resolve mountpoint for restored volume {name}")
    mountpoint = pathlib.Path(str(rows[0]["Mountpoint"]))
    clear_directory(mountpoint)
    safe_extract(archive, mountpoint)


def restore(args: argparse.Namespace) -> int:
    if not args.confirm_restore:
        raise BackupRestoreError("restore is destructive; pass --confirm-restore after reviewing the bundle")
    bundle = pathlib.Path(args.bundle).resolve()
    verify = verify_bundle(bundle)
    manifest = load_manifest(bundle)

    install_dir = pathlib.Path(args.install_dir).resolve()
    compose_file = pathlib.Path(args.compose_file or install_dir / "docker-compose.prod.yml").resolve()
    target_env = pathlib.Path(args.env_file or install_dir / ".env").resolve()
    if not install_dir.joinpath(".git").is_dir():
        raise BackupRestoreError(f"restore target is not a Git checkout: {install_dir}")
    if not compose_file.is_file():
        raise BackupRestoreError(f"missing production Compose file: {compose_file}")

    config_record = next((item for item in manifest["files"] if item.get("kind") == "production-config"), None)
    db_record = next((item for item in manifest["files"] if item.get("kind") == "control-plane-postgres"), None)
    static_record = next((item for item in manifest["files"] if item.get("kind") == "static-artifacts"), None)
    compose_record = next((item for item in manifest["files"] if item.get("kind") == "compose-workspaces"), None)
    if not all((config_record, db_record, static_record, compose_record)):
        raise BackupRestoreError("bundle is missing one or more mandatory restore records")

    source_sha = str(manifest.get("sourceGitSha") or "").strip()
    if args.require_matching_git_sha:
        target_sha = git_sha(install_dir)
        if target_sha != source_sha:
            raise BackupRestoreError(f"target checkout {target_sha} does not match backup source {source_sha}")

    target_env.parent.mkdir(parents=True, exist_ok=True)
    if target_env.exists():
        backup_name = target_env.with_name(target_env.name + ".pre-restore-" + str(int(time.time())))
        shutil.copy2(target_env, backup_name)
        os.chmod(backup_name, 0o600)
    shutil.copyfile(bundle / str(config_record["path"]), target_env)
    os.chmod(target_env, 0o600)
    env = parse_env_file(target_env)
    postgres_user = required_env(env, "POSTGRES_USER")
    postgres_db = required_env(env, "POSTGRES_DB")

    static_root = pathlib.Path(args.static_root)
    compose_root = pathlib.Path(args.compose_root)
    clear_directory(static_root)
    safe_extract(bundle / str(static_record["path"]), static_root)
    clear_directory(compose_root)
    safe_extract(bundle / str(compose_record["path"]), compose_root)

    volume_records = [item for item in manifest["files"] if item.get("kind") == "managed-volume"]
    for record in volume_records:
        restore_volume(bundle, record)

    ensure_postgres(install_dir, target_env, compose_file, postgres_user, postgres_db)
    restore_control_database(bundle / str(db_record["path"]), postgres_user, postgres_db)

    report = {
        "schemaVersion": 1,
        "kind": "backup-restore",
        "restoredAt": utc_now(),
        "sourceGitSha": source_sha,
        "targetInstallDir": str(install_dir),
        "bundleVerified": verify["status"] == "PASS",
        "configurationRestored": True,
        "controlPlaneDatabaseRestored": True,
        "staticArtifactsRestored": True,
        "composeWorkspacesRestored": True,
        "managedVolumesRestored": len(volume_records),
        "status": "PASS",
        "nextStep": "run deploy-to-vm.sh, verify-production.sh, then execute the fresh-VM beta acceptance checks",
    }
    report_path = pathlib.Path(args.report or bundle / "restore-report.json")
    write_json_private(report_path, report)
    print(report_path)
    return 0


def plan(args: argparse.Namespace) -> int:
    output = {
        "kind": "backup-restore-plan",
        "backupIncludes": [
            "production .env (sensitive, private bundle file)",
            "control-plane PostgreSQL custom dump",
            DEFAULT_STATIC_ROOT,
            DEFAULT_COMPOSE_ROOT,
            "volumes labeled mypaas.managed=true",
            "Compose volumes belonging to active MyPaas projects",
        ],
        "backupConsistency": "running managed-volume consumers require --quiesce-managed-containers",
        "restoreSafety": "restore requires --confirm-restore and verifies all manifest checksums before mutation",
        "runtimeAcceptance": "fresh-VM login/projects/env/routes/deployments/persistent-data/DB-Studio checks remain external evidence",
    }
    print(json.dumps(output, indent=2))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)

    backup_parser = sub.add_parser("backup", help="create a full disaster-recovery bundle")
    backup_parser.add_argument("--install-dir", default=os.getenv("MYPAAS_INSTALL_DIR", DEFAULT_INSTALL_DIR))
    backup_parser.add_argument("--env-file", default="")
    backup_parser.add_argument("--static-root", default=DEFAULT_STATIC_ROOT)
    backup_parser.add_argument("--compose-root", default=DEFAULT_COMPOSE_ROOT)
    backup_parser.add_argument("--output", default="")
    backup_parser.add_argument("--quiesce-managed-containers", action="store_true")
    backup_parser.set_defaults(func=backup)

    verify_parser = sub.add_parser("verify", help="verify bundle manifest and checksums without mutation")
    verify_parser.add_argument("--bundle", required=True)
    verify_parser.add_argument("--report", default="")

    restore_parser = sub.add_parser("restore", help="restore a verified bundle onto a prepared host")
    restore_parser.add_argument("--bundle", required=True)
    restore_parser.add_argument("--install-dir", default=os.getenv("MYPAAS_INSTALL_DIR", DEFAULT_INSTALL_DIR))
    restore_parser.add_argument("--env-file", default="")
    restore_parser.add_argument("--compose-file", default="")
    restore_parser.add_argument("--static-root", default=DEFAULT_STATIC_ROOT)
    restore_parser.add_argument("--compose-root", default=DEFAULT_COMPOSE_ROOT)
    restore_parser.add_argument("--report", default="")
    restore_parser.add_argument("--require-matching-git-sha", action="store_true", default=True)
    restore_parser.add_argument("--no-require-matching-git-sha", dest="require_matching_git_sha", action="store_false")
    restore_parser.add_argument("--confirm-restore", action="store_true")
    restore_parser.set_defaults(func=restore)

    plan_parser = sub.add_parser("plan", help="show the backup/restore contract without touching the host")
    plan_parser.set_defaults(func=plan)
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.command == "verify":
        report = verify_bundle(pathlib.Path(args.bundle).resolve())
        if args.report:
            write_json_private(pathlib.Path(args.report), report)
        else:
            print(json.dumps(report, indent=2))
        return 0
    try:
        return int(args.func(args))
    except BackupRestoreError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
