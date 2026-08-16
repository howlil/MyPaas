import importlib.util
import contextlib
import io
import json
import pathlib
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
            static_archive.parent.mkdir()
            volume_archive.parent.mkdir()
            static_source = pathlib.Path(tmp) / "static-src"
            compose_source = pathlib.Path(tmp) / "compose-src"
            volume_source = pathlib.Path(tmp) / "volume-src"
            for source, filename in (
                (static_source, "index.html"),
                (compose_source, "docker-compose.yml"),
                (volume_source, "sentinel.txt"),
            ):
                source.mkdir()
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
                    "manifest": {"minManagedVolumes": 1},
                }),
                encoding="utf-8",
            )
            args = types.SimpleNamespace(bundle=str(bundle), spec=str(spec), report="")
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(backup_restore.validate_fixture_manifest(args), 0)


if __name__ == "__main__":
    unittest.main()
