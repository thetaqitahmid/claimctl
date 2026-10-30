import datetime
import http.server
import json
import os
import pty
import select
import subprocess
import tempfile
import threading
import time
import unittest
from pathlib import Path


CLI = Path(__file__).resolve().parents[1]


class ExecPTYTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory = tempfile.TemporaryDirectory(prefix="claimctl-pty-")
        cls.binary = str(Path(cls.directory.name) / "claimctl")
        subprocess.run(["go", "build", "-o", cls.binary, "."],
                       cwd=CLI, check=True)

    @classmethod
    def tearDownClass(cls):
        cls.directory.cleanup()

    def run_command(self, script, input_bytes, expected_code):
        completed = []

        class Handler(http.server.BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def snapshot(self):
                now = datetime.datetime.now(datetime.timezone.utc)
                body = json.dumps({
                    "id": "claim", "resourceId": "gpu", "status": "active",
                    "leaseExpiresAt": (now + datetime.timedelta(
                        seconds=30)).isoformat(),
                    "hardEndTime": (now + datetime.timedelta(
                        hours=1)).isoformat(),
                }).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def do_GET(self):
                self.snapshot()

            def do_POST(self):
                self.snapshot()

            def do_PATCH(self):
                completed.append(self.path)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"{}")

        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        try:
            with tempfile.TemporaryDirectory() as config:
                pid, fd = pty.fork()
                if pid == 0:
                    os.environ["XDG_CONFIG_HOME"] = config
                    os.environ["CLAIMCTL_TOKEN"] = "synthetic"
                    os.environ["CLAIMCTL_TOKEN_FILE"] = ""
                    os.environ["CLAIMCTL_CA_CERT"] = ""
                    os.environ["CLAIMCTL_JSON"] = "false"
                    os.execv(self.binary, [
                        "claimctl", "--url",
                        f"http://127.0.0.1:{server.server_port}",
                        "--netrc=false", "exec", "gpu", "--", "sh", "-c",
                        script,
                    ])
                thread.start()
                output = b""
                sent = False
                status = None
                deadline = time.monotonic() + 5
                try:
                    while time.monotonic() < deadline:
                        ready, _, _ = select.select([fd], [], [], 0.05)
                        if ready:
                            try:
                                output += os.read(fd, 4096)
                            except OSError:
                                break
                        if b"READY" in output and not sent:
                            os.write(fd, input_bytes)
                            sent = True
                        done, state = os.waitpid(pid, os.WNOHANG)
                        if done:
                            status = state
                            break
                finally:
                    if status is None:
                        done, state = os.waitpid(pid, os.WNOHANG)
                        if not done:
                            os.kill(pid, 9)
                            _, state = os.waitpid(pid, 0)
                        status = state
                    os.close(fd)
                self.assertEqual(os.waitstatus_to_exitcode(status),
                                 expected_code, output)
                self.assertEqual(completed,
                                 ["/api/reservations/claim/complete"])
                return output
        finally:
            if thread.ident is not None:
                server.shutdown()
                thread.join()
            server.server_close()

    def test_child_can_read_foreground_terminal(self):
        output = self.run_command(
            'echo READY; read -r value; printf "RECEIVED:%s\\n" "$value"',
            b"hello\n", 0)
        self.assertIn(b"RECEIVED:hello", output)

    def test_terminal_interrupt_preserves_signal_exit(self):
        self.run_command("echo READY; sleep 30", b"\x03", 130)


if __name__ == "__main__":
    unittest.main(verbosity=2)
