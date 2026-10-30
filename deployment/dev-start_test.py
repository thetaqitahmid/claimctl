import base64
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]


class DevelopmentStartupTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        for directory in ("backend", "frontend", "deployment", "bin"):
            (self.root / directory).mkdir()
        for name in ("dev-env.sh", "dev-up.sh"):
            shutil.copy(ROOT / "deployment" / name, self.root / "deployment" / name)
        shutil.copy(ROOT / "backend/.env.example", self.root / "backend/.env.example")
        shutil.copy(ROOT / "Makefile", self.root / "Makefile")
        self.environment = dict(os.environ, PATH=f"{self.root / 'bin'}:{os.environ['PATH']}")

    def command(self, name, contents):
        path = self.root / "bin" / name
        path.write_text("#!/usr/bin/env bash\nset -eu\n" + contents)
        path.chmod(0o700)

    def run_make(self, target):
        return subprocess.run(
            ["make", target], cwd=self.root, env=self.environment,
            text=True, capture_output=True, timeout=15,
        )

    def config_values(self):
        result = subprocess.run(
            ["bash", "-ec", 'source backend/.env; printf "%s\\n" "$DATABASE_URL" "$ADMIN_PASSWORD" "$APP_ENCRYPTION_KEY" "${OIDC_ISSUER:-}"'],
            cwd=self.root, text=True, capture_output=True, check=True,
        )
        return result.stdout.splitlines()

    def test_fresh_worktree_gets_private_valid_configuration(self):
        result = self.run_make("dev_init")
        self.assertEqual(result.returncode, 0, result.stderr)
        database, password, key, issuer = self.config_values()
        self.assertEqual(database, "postgresql://devuser:devpass@127.0.0.1:5432/devdb?sslmode=disable")
        self.assertRegex(password, r"^Dev1![0-9a-f]{40}$")
        self.assertEqual(len(base64.b64decode(key, validate=True)), 32)
        self.assertEqual(issuer, "")
        self.assertEqual((self.root / "backend/.env").stat().st_mode & 0o777, 0o600)
        self.assertNotIn(password, result.stdout + result.stderr)
        self.assertNotIn(key, result.stdout + result.stderr)
        self.assertEqual(list((self.root / "backend").glob(".env.*")), [self.root / "backend/.env.example"])

    def test_existing_config_is_never_overwritten(self):
        path = self.root / "backend/.env"
        path.write_text("ADMIN_PASSWORD='existing password'\n")
        result = self.run_make("dev_init")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(path.read_text(), "ADMIN_PASSWORD='existing password'\n")

    def test_generated_credentials_survive_repeated_startup(self):
        self.assertEqual(self.run_make("dev_init").returncode, 0)
        initial = (self.root / "backend/.env").read_bytes()
        self.assertEqual(self.run_make("dev_init").returncode, 0)
        self.assertEqual((self.root / "backend/.env").read_bytes(), initial)

    def test_database_start_waits_until_healthy(self):
        self.command("docker", 'printf "%s\\n" "$*" > docker-arguments\n')
        result = self.run_make("db_up")
        self.assertEqual(result.returncode, 0, result.stderr)
        arguments = (self.root / "docker-arguments").read_text()
        self.assertIn("--wait --wait-timeout 90", arguments)
        self.assertTrue((self.root / "backend/.env").exists())

    def test_manual_migrations_use_the_configured_database(self):
        (self.root / "backend/.env").write_text('DATABASE_URL="postgresql://custom:password@localhost:55432/custom"\n')
        self.command("migrate", 'printf "%s\\n" "$*" > migrate-arguments\n')
        result = self.run_make("migrate_up")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("localhost:55432/custom", (self.root / "backend/migrate-arguments").read_text())
        self.assertNotIn("5432/devdb", (self.root / "backend/migrate-arguments").read_text())

    def prepare_services(self, backend_body, frontend_body):
        frontend = self.root / "frontend"
        (frontend / "node_modules").mkdir()
        for name in ("package.json", "package-lock.json", "node_modules/.package-lock.json"):
            (frontend / name).write_text("{}")
        self.command("docker", "exit 0\n")
        self.command("go", 'cp ../backend-fixture main\nchmod +x main\n')
        (self.root / "backend-fixture").write_text("#!/usr/bin/env bash\n" + backend_body)
        self.command("node", frontend_body)

    def test_backend_failure_stops_frontend_and_reports_failure(self):
        self.prepare_services(
            'for attempt in {1..100}; do [[ -f ../frontend-started ]] && break; sleep 0.02; done\nexit 42\n',
            'touch ../frontend-started\ntrap \'touch ../frontend-stopped; exit 0\' TERM\nwhile true; do sleep 0.1; done\n',
        )
        result = self.run_make("dev_up")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("42", result.stderr)
        self.assertTrue((self.root / "frontend-stopped").exists())

    def test_frontend_failure_stops_backend_and_reports_failure(self):
        self.prepare_services(
            'touch ../backend-started\ntrap \'touch ../backend-stopped; exit 0\' TERM\nwhile true; do sleep 0.1; done\n',
            'for attempt in {1..100}; do [[ -f ../backend-started ]] && break; sleep 0.02; done\nexit 43\n',
        )
        result = self.run_make("dev_up")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("43", result.stderr)
        self.assertTrue((self.root / "backend-stopped").exists())

    def test_signal_stops_both_services(self):
        self.prepare_services(
            'touch ../backend-started\ntrap \'touch ../backend-stopped; exit 0\' TERM\nwhile true; do sleep 0.1; done\n',
            'touch ../frontend-started\ntrap \'touch ../frontend-stopped; exit 0\' TERM\nwhile true; do sleep 0.1; done\n',
        )
        process = subprocess.Popen(
            ["bash", "deployment/dev-up.sh"], cwd=self.root,
            env=self.environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        )
        try:
            import time
            deadline = time.monotonic() + 10
            while not all((self.root / name).exists() for name in ("backend-started", "frontend-started")):
                if process.poll() is not None or time.monotonic() > deadline:
                    self.fail("Development services did not start")
                time.sleep(0.02)
            process.send_signal(signal.SIGTERM)
            process.communicate(timeout=10)
            self.assertEqual(process.returncode, 143)
            self.assertTrue((self.root / "backend-stopped").exists())
            self.assertTrue((self.root / "frontend-stopped").exists())
        finally:
            if process.poll() is None:
                process.kill()
                process.communicate(timeout=10)


if __name__ == "__main__":
    unittest.main()
