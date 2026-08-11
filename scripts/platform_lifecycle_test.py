import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
UNINSTALL = ROOT_DIR / "scripts" / "uninstall-vm.sh"
MIGRATE_IMPORT = ROOT_DIR / "scripts" / "migrate-import.sh"


class PlatformLifecycleTest(unittest.TestCase):
    def test_uninstall_cleans_all_owned_networks_and_statd(self) -> None:
        content = UNINSTALL.read_text(encoding="utf-8")

        self.assertIn('CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"', content)
        self.assertIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"', content)
        self.assertIn('ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"', content)
        self.assertIn('for network in "$ROUTING_NETWORK" "$PROJECT_NETWORK" "$CONTROL_NETWORK"', content)
        self.assertIn('REMOVE_STATD="${REMOVE_STATD:-true}"', content)
        self.assertIn("systemctl disable --now mypaas-statd", content)
        self.assertIn("/etc/systemd/system/mypaas-statd.service", content)
        self.assertIn("/usr/local/bin/mypaas-statd", content)
        self.assertNotIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-prod}"', content)

    def test_migration_import_provisions_all_external_networks(self) -> None:
        content = MIGRATE_IMPORT.read_text(encoding="utf-8")

        self.assertIn('CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"', content)
        self.assertIn('PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"', content)
        self.assertIn('ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"', content)
        self.assertIn("CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct", content)
        self.assertIn('for network in "$CONTROL_NETWORK" "$PROJECT_NETWORK" "$ROUTING_NETWORK"', content)
        self.assertIn("--env-file .env up -d postgres", content)
        self.assertIn("--env-file .env up -d", content)
        self.assertNotIn('NETWORK="${PROJECT_NETWORK:-mypaas-prod}"', content)


if __name__ == "__main__":
    unittest.main()
