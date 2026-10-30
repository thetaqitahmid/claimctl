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

    def test_proxy_trust_is_required(self):
        result = self.render()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("backend.trustedProxyCIDRs is required", result.stderr)

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
