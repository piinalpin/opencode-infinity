"""Regresi lifecycle runner; Docker tidak diperlukan untuk kasus kegagalan cleanup."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from run import Run, finish


class CleanupTests(unittest.TestCase):
    def test_cleanup_failure_still_removes_secret_and_records_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            secret = work / "admin"
            secret.write_text("synthetic-ephemeral-password")
            run = Run(work)
            report = {"status": "passed", "checks": run.records}
            with patch.object(run, "command", side_effect=RuntimeError("compose-cleanup_failed")), patch("run.subprocess.run"):
                with self.assertRaises(RuntimeError):
                    finish(run, ["docker", "compose", "-p", "fixture"], "fixture:unique", secret, report, True)
            self.assertFalse(secret.exists(), "secret_remains_after_cleanup_failure")
            self.assertEqual(json.loads((work / "checks.json").read_text())["status"], "failed")


if __name__ == "__main__":
    unittest.main()
