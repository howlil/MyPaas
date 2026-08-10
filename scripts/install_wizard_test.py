import importlib.util
import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
WIZARD_PATH = ROOT_DIR / "scripts" / "install-wizard.py"
RUNNER_PATH = ROOT_DIR / "scripts" / "run-install-wizard.sh"
SPEC = importlib.util.spec_from_file_location("install_wizard", WIZARD_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("unable to load install wizard")
WIZARD = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(WIZARD)


class InstallConfigTest(unittest.TestCase):
    def test_wizard_uses_docker_bind_host_for_caddy_upstream(self) -> None:
        values = dict(WIZARD.DEFAULTS)
        values.update(
            {
                "PUBLIC_DOMAIN": "mypaas.example.com",
                "OWNER_EMAIL": "owner@example.com",
                "GITHUB_CLIENT_ID": "client-id",
                "GITHUB_CLIENT_SECRET": "client-secret",
                "CLOUDFLARE_TUNNEL_TOKEN": "tunnel-token",
                "DOCKER_BIND_HOST": "172.18.0.1",
            }
        )

        content = WIZARD.build_env(values)

        self.assertIn("DOCKER_BIND_HOST=172.18.0.1", content)
        self.assertIn("CADDY_UPSTREAM_HOST=172.18.0.1", content)

    def test_terminal_installer_uses_detected_bind_host_for_caddy(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("CADDY_UPSTREAM_HOST=$docker_bind_host", installer)

    def test_podman_installer_installs_catatonit_for_buildkit(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn("! command_exists catatonit", installer)
        self.assertIn("podman catatonit docker-ce-cli", installer)

    def test_installer_installs_statd_host_daemon_by_default(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn('INSTALL_STATD="${INSTALL_STATD:-true}"', installer)
        self.assertIn("https://github.com/nabilrn/mypaas-statd.git", installer)
        self.assertIn("systemctl enable --now mypaas-statd", installer)
        self.assertIn("STATD_SOCKET=/run/mypaas/statd.sock", installer)

        values = dict(WIZARD.DEFAULTS)
        values.update(
            {
                "PUBLIC_DOMAIN": "mypaas.example.com",
                "OWNER_EMAIL": "owner@example.com",
                "GITHUB_CLIENT_ID": "client-id",
                "GITHUB_CLIENT_SECRET": "client-secret",
                "CLOUDFLARE_TUNNEL_TOKEN": "tunnel-token",
            }
        )
        content = WIZARD.build_env(values)
        self.assertIn("STATD_SOCKET=/run/mypaas/statd.sock", content)

    def test_production_compose_avoids_nested_env_expansion(self) -> None:
        compose = (ROOT_DIR / "docker-compose.prod.yml").read_text(encoding="utf-8")

        self.assertIn(
            "CADDY_UPSTREAM_HOST: ${CADDY_UPSTREAM_HOST:-host.docker.internal}",
            compose,
        )
        self.assertNotIn("${CADDY_UPSTREAM_HOST:-${", compose)

    def test_installer_enables_temporary_public_wizard_by_default(self) -> None:
        installer = (ROOT_DIR / "scripts" / "install-vm.sh").read_text(encoding="utf-8")

        self.assertIn('WIZARD_PUBLIC_TUNNEL="${WIZARD_PUBLIC_TUNNEL:-true}"', installer)
        self.assertIn('bash "$ROOT_DIR/scripts/run-install-wizard.sh"', installer)

    def test_wizard_runner_uses_ephemeral_cloudflare_tunnel_and_cleanup(self) -> None:
        runner = RUNNER_PATH.read_text(encoding="utf-8")

        self.assertIn("cloudflare/cloudflared:latest", runner)
        self.assertIn("--network host", runner)
        self.assertIn("trycloudflare\\.com", runner)
        self.assertIn("trap cleanup EXIT INT TERM", runner)
        self.assertIn("SSH fallback", runner)


if __name__ == "__main__":
    unittest.main()
