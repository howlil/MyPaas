import importlib.util
import io
import json
import pathlib
import tarfile
import tempfile
import unittest


SCRIPT = pathlib.Path(__file__).with_name("backup-restore.py")
SPEC = importlib.util.spec_from_file_location("backup_restore", SCRIPT)
backup_restore = importlib.util.module_from_spec(SPEC)
assert SPEC and SPEC.loader
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


if __name__ == "__main__":
    unittest.main()
