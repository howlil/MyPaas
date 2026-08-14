import os
import pathlib
import subprocess
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[1]
SCRIPT = REPO_ROOT / "scripts" / "docker-retention.sh"


class DockerRetentionScriptTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.tmp.name)
        (self.root / ".git").mkdir()
        self.log = self.root / "docker.log"
        self.fake_docker = self.root / "docker"
        self.fake_docker.write_text(
            "#!/usr/bin/env bash\n"
            "printf '%s\\n' \"$*\" >> \"$FAKE_DOCKER_LOG\"\n"
            "case \"$1 $2\" in\n"
            "  'system df') echo 'TYPE TOTAL ACTIVE SIZE RECLAIMABLE' ;;\n"
            "  'image ls') echo 'REPOSITORY TAG IMAGE ID' ;;\n"
            "  'builder du') echo 'ID RECLAIMABLE SIZE' ;;\n"
            "  'ps --format') echo 'mypaas-demo image running' ;;\n"
            "esac\n",
            encoding="utf-8",
        )
        self.fake_docker.chmod(0o755)

    def tearDown(self):
        self.tmp.cleanup()

    def run_script(self, *args):
        env = os.environ.copy()
        env.update(
            {
                "DOCKER_BIN": str(self.fake_docker),
                "FAKE_DOCKER_LOG": str(self.log),
                "MYPAAS_INSTALL_DIR": str(self.root),
                "IMAGE_CLEANUP_UNTIL": "168h",
            }
        )
        return subprocess.run(
            ["bash", str(SCRIPT), *args],
            env=env,
            text=True,
            capture_output=True,
            check=False,
        )

    def docker_calls(self):
        if not self.log.exists():
            return []
        return self.log.read_text(encoding="utf-8").splitlines()

    def test_default_is_dry_run_and_does_not_prune(self):
        completed = self.run_script()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        calls = self.docker_calls()
        self.assertFalse(any(call.startswith("image prune") for call in calls))
        self.assertFalse(any(call.startswith("builder prune") for call in calls))
        self.assertIn("No data was deleted", completed.stdout)

    def test_apply_runs_age_scoped_image_and_builder_prune(self):
        completed = self.run_script("--apply")
        self.assertEqual(completed.returncode, 0, completed.stderr)
        calls = self.docker_calls()
        self.assertIn(
            "image prune -a -f --filter label=mypaas.managed=true --filter until=168h",
            calls,
        )
        self.assertIn("builder prune -f --filter until=168h", calls)

    def test_apply_refuses_while_update_lock_exists(self):
        (self.root / ".git" / "mypaas-update.lock").mkdir()
        completed = self.run_script("--apply")
        self.assertEqual(completed.returncode, 1)
        self.assertIn("Refusing cleanup while the MyPaas updater lock exists", completed.stderr)
        calls = self.docker_calls()
        self.assertFalse(any(call.startswith("image prune") for call in calls))
        self.assertFalse(any(call.startswith("builder prune") for call in calls))

    def test_invalid_retention_window_fails_closed(self):
        env = os.environ.copy()
        env.update(
            {
                "DOCKER_BIN": str(self.fake_docker),
                "FAKE_DOCKER_LOG": str(self.log),
                "MYPAAS_INSTALL_DIR": str(self.root),
                "IMAGE_CLEANUP_UNTIL": "all",
            }
        )
        completed = subprocess.run(
            ["bash", str(SCRIPT), "--apply"],
            env=env,
            text=True,
            capture_output=True,
            check=False,
        )
        self.assertEqual(completed.returncode, 2)
        self.assertEqual(self.docker_calls(), [])


if __name__ == "__main__":
    unittest.main()
