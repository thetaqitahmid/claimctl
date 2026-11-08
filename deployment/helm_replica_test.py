import subprocess
import unittest
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parent.parent


class HelmReplicaTest(unittest.TestCase):
    def render(self, *settings):
        command = ["helm", "template", "fixture", str(ROOT / "charts/claimctl")]
        for setting in settings:
            command.extend(["--set", setting])
        return subprocess.run(command, capture_output=True, text=True)

    def deployments(self, result):
        self.assertEqual(result.returncode, 0, result.stderr)
        return {
            item["metadata"]["name"]: item
            for item in yaml.safe_load_all(result.stdout)
            if item and item["kind"] == "Deployment"
        }

    def environment(self, deployment):
        container = deployment["spec"]["template"]["spec"]["containers"][0]
        return {entry["name"]: entry for entry in container["env"]}

    def test_pipeline_validation_commands(self):
        workflow = yaml.safe_load(
            (ROOT / ".github/workflows/helm.yml").read_text())
        steps = workflow["jobs"]["lint"]["steps"]
        for name in ["Lint chart", "Template chart"]:
            with self.subTest(step=name):
                script = next(step["run"] for step in steps
                              if step.get("name") == name)
                result = subprocess.run(
                    ["sh", "-ec", script], cwd=ROOT,
                    capture_output=True, text=True)
                self.assertEqual(result.returncode, 0,
                                 result.stdout + result.stderr)

    def test_make_test_includes_helm_validation(self):
        result = subprocess.run(
            ["make", "--dry-run", "test"], cwd=ROOT,
            capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("helm dependency build charts/claimctl", result.stdout)
        for command in ["helm lint charts/claimctl",
                        "helm template claimctl charts/claimctl"]:
            self.assertIn(
                command + " -f charts/claimctl/ci/lint-values.yaml",
                result.stdout)

    def test_proxy_trust_is_optional(self):
        result = self.render()
        backend = self.deployments(result)["fixture-claimctl-backend"]
        environment = self.environment(backend)
        self.assertEqual(environment["TRUSTED_PROXY_CIDRS"]["value"], "")

    def test_internal_backend_and_explicit_proxy_configuration(self):
        result = self.render(
            "backend.trustedProxyCIDRs=10.244.0.0/16",
            "frontend.trustedIngressCIDR=10.245.0.0/16",
            "backend.replicaCount=3",
            "ingress.enabled=true",
            "ingress.tls[0].secretName=synthetic-tls",
            "ingress.tls[0].hosts[0]=synthetic.example",
        )
        deployments = self.deployments(result)
        backend = deployments["fixture-claimctl-backend"]
        self.assertEqual(backend["spec"]["replicas"], 3)
        environment = self.environment(backend)
        self.assertEqual(environment["TRUSTED_PROXY_CIDRS"]["value"],
                         "10.244.0.0/16")
        self.assertEqual(environment["DATABASE_URL"]["value"], "")
        self.assertEqual(environment["COOKIE_SECURE"]["value"], "true")
        self.assertIn("secretKeyRef",
                      environment["APP_ENCRYPTION_KEY"]["valueFrom"])
        frontend = self.environment(deployments["fixture-claimctl-frontend"])
        self.assertEqual(frontend["BACKEND_HOST"]["value"],
                         "fixture-claimctl-backend")
        self.assertEqual(frontend["BACKEND_PORT"]["value"], "3000")
        self.assertEqual(frontend["TRUSTED_INGRESS_CIDR"]["value"],
                         "10.245.0.0/16")
        ingress = next(item for item in yaml.safe_load_all(result.stdout)
                       if item and item["kind"] == "Ingress")
        annotations = ingress["metadata"]["annotations"]
        self.assertEqual(
            annotations["nginx.ingress.kubernetes.io/proxy-buffering"], "off")
        self.assertEqual(
            annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"], "960")


if __name__ == "__main__":
    unittest.main(verbosity=2)
