import importlib.util
import json
import pathlib
import sys
import unittest

MODULE_PATH = pathlib.Path(__file__).with_name("run.py")
spec = importlib.util.spec_from_file_location("compatibility_run", MODULE_PATH)
runner = importlib.util.module_from_spec(spec)
assert spec.loader is not None
sys.modules[spec.name] = runner
spec.loader.exec_module(runner)


class CompatibilitySuiteTests(unittest.TestCase):
    def test_repository_catalog_validates_without_compose_execution(self):
        catalog = runner.load_catalog()
        self.assertEqual([], runner.validate_catalog(catalog, compose=False))
        self.assertGreaterEqual(len(catalog["applications"]), 10)

    def test_default_selection_excludes_heavy_and_blocked_entries(self):
        catalog = runner.load_catalog()
        selected = runner.selected_apps(catalog, [], include_heavy=False, include_blocked=False)
        ids = {app["id"] for app in selected}
        self.assertNotIn("immich", ids)
        self.assertNotIn("appsmith", ids)
        self.assertNotIn("minio", ids)
        self.assertIn("excalidraw", ids)
        self.assertIn("n8n", ids)

    def test_explicit_heavy_selection_can_be_enabled(self):
        catalog = runner.load_catalog()
        selected = runner.selected_apps(catalog, ["immich"], include_heavy=True, include_blocked=False)
        self.assertEqual(["immich"], [app["id"] for app in selected])

    def test_unknown_id_is_rejected(self):
        catalog = runner.load_catalog()
        with self.assertRaises(runner.SuiteError):
            runner.selected_apps(catalog, ["does-not-exist"], False, False)

    def test_project_payload_uses_core_repo_for_local_manifests(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "n8n")
        payload = runner.project_payload(catalog, app, "compat-n8n-test")
        self.assertEqual("git", payload["sourceType"])
        self.assertEqual("compose", payload["deployMode"])
        self.assertEqual("https://github.com/nabilrn/MyPaas.git", payload["repoUrl"])
        self.assertEqual("compatibility/manifests/n8n", payload["baseDirectory"])

    def test_registry_payload_does_not_require_repository(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "excalidraw")
        payload = runner.project_payload(catalog, app, "compat-excalidraw")
        self.assertEqual("registry", payload["sourceType"])
        self.assertEqual("", payload["repoUrl"])
        self.assertEqual("image", payload["deployMode"])
        self.assertTrue(payload["imageRef"])

    def test_invalid_duplicate_catalog_is_reported(self):
        catalog = runner.load_catalog()
        clone = json.loads(json.dumps(catalog))
        clone["applications"].append(clone["applications"][0])
        errors = runner.validate_catalog(clone)
        self.assertTrue(any("duplicate application id" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
