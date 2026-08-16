import importlib.util
import contextlib
import io
import json
import pathlib
import subprocess
import sys
import tarfile
import tempfile
import types
import unittest
from unittest import mock


SCRIPT = pathlib.Path(__file__).with_name("backup-restore.py")
SPEC = importlib.util.spec_from_file_location("backup_restore", SCRIPT)
backup_restore = importlib.util.module_from_spec(SPEC)
assert SPEC and SPEC.loader
sys.modules[SPEC.name] = backup_restore
SPEC.loader.exec_module(backup_restore)


class BackupRestoreTest(unittest.TestCase):
    def test_parse_env_file_handles_quotes_without_logging_values(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / ".env"
            path.write_text(
                "POSTGRES_USER=mypaas\n"
                "POSTGRES_PASSWORD='super-secret'\n"
                'ENCRYPTION_KEY="private-key"\n',
                encoding="utf-8",
            )
            values = backup_restore.parse_env_file(path)
            self.assertEqual(values["POSTGRES_PASSWORD"], "super-secret")
            self.assertEqual(values["ENCRYPTION_KEY"], "private-key")

    def test_classify_volume_prefers_mypaas_label(self):
        info = {
            "Name": "random-name",
            "Labels": {"mypaas.managed": "true"},
        }
        self.assertEqual(backup_restore.classify_volume(info, set()), "mypaas-managed")

    def test_classify_compose_volume_requires_active_project(self):
        info = {
            "Name": "demo_data",
            "Labels": {"com.docker.compose.project": "demo"},
        }
        self.assertEqual(backup_restore.classify_volume(info, {"demo"}), "mypaas-compose")
        self.assertIsNone(backup_restore.classify_volume(info, {"other"}))

    def test_active_project_volume_names_include_mypaas_compose_runtime_name(self):
        completed = subprocess.CompletedProcess(
            args=[],
            returncode=0,
            stdout=b"phase2-compose-1c5149\tcompose\nphase2-static-1c5149\tstatic\n",
            stderr=b"",
        )
        with mock.patch.object(backup_restore, "run", return_value=completed):
            names = backup_restore.active_project_volume_names("mypaas", "mypaas")
        self.assertIn("phase2-compose-1c5149", names)
        self.assertIn("mypaas-phase2-compose-1c5149", names)
        self.assertIn("phase2-static-1c5149", names)
        self.assertNotIn("mypaas-phase2-static-1c5149", names)

    def test_classify_compose_volume_matches_mypaas_runtime_project_label(self):
        info = {
            "Name": "compose_beta-data",
            "Labels": {"com.docker.compose.project": "mypaas-phase2-compose-1c5149"},
        }
        self.assertEqual(
            backup_restore.classify_volume(info, {"phase2-compose-1c5149", "mypaas-phase2-compose-1c5149"}),
            "mypaas-compose",
        )

    def test_verify_bundle_detects_tampering(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = pathlib.Path(tmp)
            payload = bundle / "filesystem" / "static.tar.gz"
            payload.parent.mkdir()
            payload.write_bytes(b"original")
            manifest = {
                "schemaVersion": 1,
                "kind": "mypaas-full-backup",
                "sourceGitSha": "abc123",
                "files": [backup_restore.file_record("static-artifacts", bundle, payload)],
            }
            (bundle / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            report = backup_restore.verify_bundle(bundle)
            self.assertEqual(report["status"], "PASS")
            payload.write_bytes(b"tampered")
            with self.assertRaises(backup_restore.BackupRestoreError):
                backup_restore.verify_bundle(bundle)

    def test_safe_extract_rejects_parent_traversal(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            archive_path = root / "bad.tar.gz"
            data = b"escape"
            with tarfile.open(archive_path, "w:gz") as archive:
                member = tarfile.TarInfo("../escape.txt")
                member.size = len(data)
                archive.addfile(member, io.BytesIO(data))
            destination = root / "restore"
            with self.assertRaises(backup_restore.BackupRestoreError):
                backup_restore.safe_extract(archive_path, destination)
            self.assertFalse((root / "escape.txt").exists())

    def test_manifest_record_contains_checksum_not_file_contents(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = pathlib.Path(tmp)
            private = bundle / "private" / "config.env"
            private.parent.mkdir()
            private.write_text("TOKEN=do-not-report\n", encoding="utf-8")
            record = backup_restore.file_record("production-config", bundle, private)
            encoded = json.dumps(record)
            self.assertNotIn("do-not-report", encoded)
            self.assertEqual(record["path"], "private/config.env")
            self.assertEqual(len(record["sha256"]), 64)

    def test_safe_member_destination_accepts_normal_relative_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            destination = backup_restore._safe_member_destination(root, "nested/file.txt")
            self.assertTrue(str(destination).startswith(str(root)))

    def test_source_preflight_fails_when_required_fixtures_are_missing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            env_file = root / ".env"
            env_file.write_text(
                "POSTGRES_USER=mypaas\nPOSTGRES_DB=mypaas\nJWT_SECRET=test-secret\nPUBLIC_DOMAIN=example.test\n",
                encoding="utf-8",
            )
            spec = root / "fixtures.json"
            spec.write_text("{}", encoding="utf-8")
            args = types.SimpleNamespace(
                spec=str(spec),
                install_dir=str(root),
                env_file=str(env_file),
                static_root=str(root / "static"),
                compose_root=str(root / "compose"),
                report="",
            )
            with mock.patch.object(backup_restore, "issue_internal_access_token", return_value="token"), \
                 mock.patch.object(backup_restore, "git_sha", return_value="abc123"), \
                 contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.preflight_source_fixtures(args), 1)

    def test_source_preflight_fails_before_backup_when_static_route_is_unhealthy(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            env_file = root / ".env"
            env_file.write_text(
                "POSTGRES_USER=mypaas\nPOSTGRES_DB=mypaas\nJWT_SECRET=test-secret\nPUBLIC_DOMAIN=example.test\n",
                encoding="utf-8",
            )
            static_root = root / "static"
            sentinel = static_root / "project-1" / "index.html"
            sentinel.parent.mkdir(parents=True)
            sentinel.write_text("STATIC-SENTINEL", encoding="utf-8")
            spec = root / "fixtures.json"
            spec.write_text(
                json.dumps({
                    "routeTimeoutSeconds": 0,
                    "static": {
                        "projectName": "static-fixture",
                        "sentinelPath": "index.html",
                        "expectedContent": "STATIC-SENTINEL",
                    }
                }),
                encoding="utf-8",
            )
            args = types.SimpleNamespace(
                spec=str(spec),
                install_dir=str(root),
                env_file=str(env_file),
                static_root=str(static_root),
                compose_root=str(root / "compose"),
                report="",
            )
            project = {
                "id": "project-1",
                "name": "static-fixture",
                "subdomain": "static-fixture",
                "deploy_mode": "static",
                "status": "running",
            }
            with mock.patch.object(backup_restore, "project_by_name", return_value=project), \
                 mock.patch.object(backup_restore, "route_body", side_effect=backup_restore.BackupRestoreError("HTTP 502")), \
                 mock.patch.object(backup_restore, "issue_internal_access_token", return_value="token"), \
                 mock.patch.object(backup_restore, "git_sha", return_value="abc123"), \
                 contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.preflight_source_fixtures(args), 1)

    def test_validate_fixture_manifest_rejects_empty_archives_and_missing_volume(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = pathlib.Path(tmp) / "bundle"
            bundle.mkdir()
            static_archive = bundle / "filesystem" / "static.tar.gz"
            compose_archive = bundle / "filesystem" / "compose.tar.gz"
            static_archive.parent.mkdir()
            with tarfile.open(static_archive, "w:gz"):
                pass
            with tarfile.open(compose_archive, "w:gz"):
                pass
            manifest = {
                "schemaVersion": 1,
                "kind": "mypaas-full-backup",
                "sourceGitSha": "abc123",
                "projectCount": 4,
                "managedVolumeCount": 0,
                "files": [
                    backup_restore.file_record("static-artifacts", bundle, static_archive),
                    backup_restore.file_record("compose-workspaces", bundle, compose_archive),
                ],
            }
            (bundle / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            spec = pathlib.Path(tmp) / "fixtures.json"
            spec.write_text(
                json.dumps({
                    "persistentVolume": {"volumeName": "fixture_data"},
                    "composeDatabase": {"volumeName": "compose_beta-data"},
                    "manifest": {"minManagedVolumes": 1},
                }),
                encoding="utf-8",
            )
            args = types.SimpleNamespace(bundle=str(bundle), spec=str(spec), report="")
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.validate_fixture_manifest(args), 1)

    def test_validate_fixture_manifest_rejects_unrelated_managed_volume_for_compose_db(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = pathlib.Path(tmp) / "bundle"
            bundle.mkdir()
            static_archive = bundle / "filesystem" / "static.tar.gz"
            compose_archive = bundle / "filesystem" / "compose.tar.gz"
            unrelated_archive = bundle / "volumes" / "phase2-managed-volume-1c5149.tar.gz"
            static_source = pathlib.Path(tmp) / "static-src"
            compose_source = pathlib.Path(tmp) / "compose-src"
            unrelated_source = pathlib.Path(tmp) / "unrelated-volume-src"
            for source, filename in (
                (static_source, "index.html"),
                (compose_source, "docker-compose.yml"),
                (unrelated_source, "sentinel.txt"),
            ):
                source.mkdir(parents=True)
                (source / filename).write_text("sentinel", encoding="utf-8")
            backup_restore.archive_directory(static_source, static_archive)
            backup_restore.archive_directory(compose_source, compose_archive)
            backup_restore.archive_directory(unrelated_source, unrelated_archive)
            manifest = {
                "schemaVersion": 1,
                "kind": "mypaas-full-backup",
                "sourceGitSha": "abc123",
                "projectCount": 4,
                "managedVolumeCount": 1,
                "files": [
                    backup_restore.file_record("static-artifacts", bundle, static_archive),
                    backup_restore.file_record("compose-workspaces", bundle, compose_archive),
                    backup_restore.file_record("managed-volume", bundle, unrelated_archive, volumeName="phase2-managed-volume-1c5149"),
                ],
            }
            (bundle / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            spec = pathlib.Path(tmp) / "fixtures.json"
            spec.write_text(
                json.dumps({
                    "persistentVolume": {"volumeName": "phase2-managed-volume-1c5149"},
                    "composeDatabase": {"volumeName": "compose_beta-data"},
                    "manifest": {"minManagedVolumes": 1},
                }),
                encoding="utf-8",
            )
            args = types.SimpleNamespace(bundle=str(bundle), spec=str(spec), report="")
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.validate_fixture_manifest(args), 1)

    def test_validate_fixture_manifest_requires_expected_compose_db_volume_in_spec(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = pathlib.Path(tmp) / "bundle"
            bundle.mkdir()
            static_archive = bundle / "filesystem" / "static.tar.gz"
            compose_archive = bundle / "filesystem" / "compose.tar.gz"
            volume_archive = bundle / "volumes" / "fixture_data.tar.gz"
            static_source = pathlib.Path(tmp) / "static-src"
            compose_source = pathlib.Path(tmp) / "compose-src"
            volume_source = pathlib.Path(tmp) / "volume-src"
            for source, filename in (
                (static_source, "index.html"),
                (compose_source, "docker-compose.yml"),
                (volume_source, "sentinel.txt"),
            ):
                source.mkdir(parents=True)
                (source / filename).write_text("sentinel", encoding="utf-8")
            backup_restore.archive_directory(static_source, static_archive)
            backup_restore.archive_directory(compose_source, compose_archive)
            backup_restore.archive_directory(volume_source, volume_archive)
            manifest = {
                "schemaVersion": 1,
                "kind": "mypaas-full-backup",
                "sourceGitSha": "abc123",
                "projectCount": 4,
                "managedVolumeCount": 1,
                "files": [
                    backup_restore.file_record("static-artifacts", bundle, static_archive),
                    backup_restore.file_record("compose-workspaces", bundle, compose_archive),
                    backup_restore.file_record("managed-volume", bundle, volume_archive, volumeName="fixture_data"),
                ],
            }
            (bundle / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            spec = pathlib.Path(tmp) / "fixtures.json"
            spec.write_text(
                json.dumps({
                    "persistentVolume": {"volumeName": "fixture_data"},
                    "composeDatabase": {"projectName": "phase2-compose-1c5149"},
                    "manifest": {"minManagedVolumes": 1},
                }),
                encoding="utf-8",
            )
            args = types.SimpleNamespace(bundle=str(bundle), spec=str(spec), report="")
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.validate_fixture_manifest(args), 1)

    def test_validate_fixture_manifest_passes_when_archives_and_volume_are_present(self):
        with tempfile.TemporaryDirectory() as tmp:
            bundle = pathlib.Path(tmp) / "bundle"
            bundle.mkdir()
            static_archive = bundle / "filesystem" / "static.tar.gz"
            compose_archive = bundle / "filesystem" / "compose.tar.gz"
            volume_archive = bundle / "volumes" / "fixture_data.tar.gz"
            compose_volume_archive = bundle / "volumes" / "compose_beta-data.tar.gz"
            static_archive.parent.mkdir()
            volume_archive.parent.mkdir()
            static_source = pathlib.Path(tmp) / "static-src"
            compose_source = pathlib.Path(tmp) / "compose-src"
            volume_source = pathlib.Path(tmp) / "volume-src"
            compose_volume_source = pathlib.Path(tmp) / "compose-volume-src"
            for source, filename in (
                (static_source, "index.html"),
                (compose_source, "docker-compose.yml"),
                (volume_source, "sentinel.txt"),
                (compose_volume_source, "pgdata"),
            ):
                source.mkdir()
                (source / filename).write_text("sentinel", encoding="utf-8")
            backup_restore.archive_directory(static_source, static_archive)
            backup_restore.archive_directory(compose_source, compose_archive)
            backup_restore.archive_directory(volume_source, volume_archive)
            backup_restore.archive_directory(compose_volume_source, compose_volume_archive)
            manifest = {
                "schemaVersion": 1,
                "kind": "mypaas-full-backup",
                "sourceGitSha": "abc123",
                "projectCount": 4,
                "managedVolumeCount": 2,
                "files": [
                    backup_restore.file_record("static-artifacts", bundle, static_archive),
                    backup_restore.file_record("compose-workspaces", bundle, compose_archive),
                    backup_restore.file_record("managed-volume", bundle, volume_archive, volumeName="fixture_data"),
                    backup_restore.file_record("managed-volume", bundle, compose_volume_archive, volumeName="compose_beta-data"),
                ],
            }
            (bundle / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            spec = pathlib.Path(tmp) / "fixtures.json"
            spec.write_text(
                json.dumps({
                    "persistentVolume": {"volumeName": "fixture_data"},
                    "composeDatabase": {"volumeName": "compose_beta-data"},
                    "manifest": {"minManagedVolumes": 2},
                }),
                encoding="utf-8",
            )
            args = types.SimpleNamespace(bundle=str(bundle), spec=str(spec), report="")
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.validate_fixture_manifest(args), 0)

    def test_restore_volume_preserves_compose_volume_identity_and_data(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            bundle = root / "bundle"
            source = root / "source"
            mountpoint = root / "docker-volumes" / "compose_beta-data" / "_data"
            source.mkdir()
            (source / "PG_VERSION").write_text("16\n", encoding="utf-8")
            archive = bundle / "volumes" / "compose_beta-data.tar.gz"
            backup_restore.archive_directory(source, archive)
            record = backup_restore.file_record("managed-volume", bundle, archive, volumeName="compose_beta-data")
            calls = []

            def fake_run(args, **kwargs):
                calls.append(args)
                return subprocess.CompletedProcess(args=args, returncode=0, stdout=b"", stderr=b"")

            def fake_docker_json(args):
                self.assertEqual(args, ["volume", "inspect", "compose_beta-data"])
                return [{"Mountpoint": str(mountpoint)}]

            with mock.patch.object(backup_restore, "run", side_effect=fake_run), \
                 mock.patch.object(backup_restore, "docker_json", side_effect=fake_docker_json):
                backup_restore.restore_volume(bundle, record)

            self.assertIn(["docker", "volume", "create", "--label", "mypaas.managed=true", "compose_beta-data"], calls)
            self.assertEqual((mountpoint / "PG_VERSION").read_text(encoding="utf-8"), "16\n")

    def test_recreate_api_if_present_forces_startup_reconciliation_after_db_restore(self):
        calls = []

        def fake_run(args, **kwargs):
            calls.append(args)
            if args[-3:] == ["--all", "-q", "api"]:
                return subprocess.CompletedProcess(args=args, returncode=0, stdout=b"mypaas-api\n", stderr=b"")
            return subprocess.CompletedProcess(args=args, returncode=0, stdout=b"", stderr=b"")

        with mock.patch.object(backup_restore, "run", side_effect=fake_run):
            recreated = backup_restore.recreate_api_if_present(pathlib.Path("/opt/mypaas"), pathlib.Path(".env"), pathlib.Path("docker-compose.prod.yml"))

        self.assertTrue(recreated)
        self.assertIn(
            ["docker", "compose", "-f", "docker-compose.prod.yml", "--env-file", ".env", "ps", "--all", "-q", "api"],
            calls,
        )
        self.assertIn(
            ["docker", "compose", "-f", "docker-compose.prod.yml", "--env-file", ".env", "up", "-d", "--force-recreate", "api"],
            calls,
        )

    def test_recreate_api_if_present_detects_stopped_api_container(self):
        calls = []

        def fake_run(args, **kwargs):
            calls.append(args)
            if args[-3:] == ["--all", "-q", "api"]:
                return subprocess.CompletedProcess(args=args, returncode=0, stdout=b"stopped-api-container\n", stderr=b"")
            return subprocess.CompletedProcess(args=args, returncode=0, stdout=b"", stderr=b"")

        with mock.patch.object(backup_restore, "run", side_effect=fake_run):
            self.assertTrue(backup_restore.recreate_api_if_present(pathlib.Path("/opt/mypaas"), pathlib.Path(".env"), pathlib.Path("docker-compose.prod.yml")))

        self.assertIn(
            ["docker", "compose", "-f", "docker-compose.prod.yml", "--env-file", ".env", "ps", "--all", "-q", "api"],
            calls,
        )
        self.assertIn(
            ["docker", "compose", "-f", "docker-compose.prod.yml", "--env-file", ".env", "up", "-d", "--force-recreate", "api"],
            calls,
        )

    def test_recreate_api_if_present_raises_when_compose_ps_fails(self):
        def fake_run(args, **kwargs):
            if args[-3:] == ["--all", "-q", "api"]:
                return subprocess.CompletedProcess(args=args, returncode=1, stdout=b"", stderr=b"compose config error")
            self.fail("force recreate should not run after failed compose ps inspection")

        with mock.patch.object(backup_restore, "run", side_effect=fake_run):
            with self.assertRaises(backup_restore.BackupRestoreError):
                backup_restore.recreate_api_if_present(pathlib.Path("/opt/mypaas"), pathlib.Path(".env"), pathlib.Path("docker-compose.prod.yml"))


if __name__ == "__main__":
    unittest.main()
