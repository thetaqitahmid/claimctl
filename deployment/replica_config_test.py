import base64
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parent.parent


class ReplicaConfigTest(unittest.TestCase):
    def compose(self, file_name, overrides):
        with tempfile.TemporaryDirectory(prefix="claimctl-compose-") as temp:
            fixture = Path(temp)
            env_file = fixture / "synthetic.env"
            env_file.write_text(
                "DATABASE_URL=postgres://wrong@127.0.0.1:1/wrong\n"
                "DB_USER=wrong\nDB_PASSWORD=wrong\nDB_NAME=wrong\n"
            )
            config = fixture / "compose.yaml"
            source = (ROOT / file_name).read_text()
            config.write_text(source.replace("./backend/.env", str(env_file)))
            env = {
                "PATH": os.environ["PATH"],
                "HOME": os.environ["HOME"],
                "DOCKER_CONFIG": os.environ.get(
                    "DOCKER_CONFIG", str(Path.home() / ".docker")
                ),
            }
            env.update(overrides)
            return subprocess.run(
                ["docker", "compose", "--project-directory", str(ROOT),
                 "--env-file", str(env_file), "-f", str(config),
                 "config", "--format", "json"],
                env=env, capture_output=True, text=True, check=False,
            )

    def test_production_credentials_and_internal_url_precedence(self):
        password = "synthetic@:/?#%+ &$password"
        result = self.compose("docker-compose.prod.yaml", {
            "DB_USER": "synthetic", "DB_PASSWORD": password,
            "DB_NAME": "synthetic",
            "APP_ENCRYPTION_KEY": base64.b64encode(bytes(32)).decode(),
            "TRUSTED_PROXY_CIDRS": "172.30.0.0/24",
            "POSTGRES_PGDATA": "/var/lib/postgresql/18/docker",
        })
        self.assertEqual(result.returncode, 0, result.stderr)
        services = json.loads(result.stdout)["services"]
        self.assertNotIn("ports", services["backend"])
        backend = services["backend"]["environment"]
        postgres = services["db"]["environment"]
        self.assertEqual(backend["DATABASE_URL"], "")
        self.assertEqual(backend["DB_HOST"], "db")
        self.assertEqual(backend["DB_PASSWORD"].replace("$$", "$"), password)
        self.assertEqual(postgres["POSTGRES_PASSWORD"], backend["DB_PASSWORD"])
        self.check_password_runtime_round_trip(backend, password)
        self.assertEqual(backend["DB_USER"], postgres["POSTGRES_USER"])
        self.assertEqual(backend["DB_NAME"], postgres["POSTGRES_DB"])
        self.assertEqual(postgres["PGDATA"], "/var/lib/postgresql/18/docker")
        self.assertIn("/var/lib/postgresql", [
            volume["target"] for volume in services["db"]["volumes"]
        ])
        self.assertEqual(services["db"]["entrypoint"],
                         ["/usr/local/bin/claimctl-postgres-preflight"])

    def check_password_runtime_round_trip(self, environment, password):
        with tempfile.TemporaryDirectory(prefix="claimctl-compose-runtime-") as temp:
            config = Path(temp) / "compose.json"
            config.write_text(json.dumps({"services": {"probe": {
                "image": "nginx:alpine", "network_mode": "none",
                "environment": environment,
                "entrypoint": ["sh", "-c", 'printf "%s" "$$DB_PASSWORD"'],
            }}}))
            result = subprocess.run([
                "docker", "--context", "default", "compose", "-f", str(config), "run", "--rm",
                "--no-deps", "-T", "probe",
            ], capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, password)

    def test_production_requires_explicit_secrets_and_layout(self):
        valid = {
            "DB_PASSWORD": "synthetic",
            "APP_ENCRYPTION_KEY": base64.b64encode(bytes(32)).decode(),
            "TRUSTED_PROXY_CIDRS": "172.30.0.0/24",
            "POSTGRES_PGDATA": "/var/lib/postgresql/18/docker",
        }
        for missing in valid:
            with self.subTest(missing=missing):
                overrides = dict(valid)
                overrides[missing] = ""
                result = self.compose("docker-compose.prod.yaml", overrides)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(missing, result.stderr)

    def test_development_uses_disposable_port_and_safe_layout(self):
        result = self.compose("docker-compose-dev.yaml", {})
        self.assertEqual(result.returncode, 0, result.stderr)
        services = json.loads(result.stdout)["services"]
        self.assertEqual(services["backend"]["environment"]["DATABASE_URL"], "")
        postgres = services["db"]
        self.assertEqual(postgres["environment"]["PGDATA"],
                         "/var/lib/postgresql/18/docker")
        port = postgres["ports"][0]
        self.assertEqual(str(port["published"]), "55432")
        self.assertEqual(port["host_ip"], "127.0.0.1")

    def test_nginx_syntax_and_sse_settings(self):
        template = ROOT / "frontend/nginx.conf.template"
        source = template.read_text()
        for setting in ["proxy_buffering off;", "proxy_cache off;",
                        "proxy_read_timeout 960s;",
                        "X-Forwarded-Proto $external_scheme;"]:
            self.assertIn(setting, source)
        command = (
            "envsubst '$BACKEND_HOST $BACKEND_PORT $TRUSTED_INGRESS_CIDR' "
            "< /tmp/default.conf.template > /tmp/default.conf; "
            "printf 'events {}\\nhttp { include /tmp/default.conf; }\\n' "
            "> /tmp/nginx.conf; nginx -t -c /tmp/nginx.conf"
        )
        result = subprocess.run([
            "docker", "--context", "default", "run", "--rm", "--network", "none",
            "--entrypoint", "sh", "-e", "BACKEND_HOST=127.0.0.1",
            "-e", "BACKEND_PORT=3000", "-e",
            "TRUSTED_INGRESS_CIDR=127.0.0.1/32",
            "-v", str(template) + ":/tmp/default.conf.template:ro",
            "nginx:alpine", "-c", command,
        ], capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("test is successful", result.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
