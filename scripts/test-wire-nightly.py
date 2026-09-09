#!/usr/bin/env python3
"""Exercise the nightly runner against real temporary Git repositories."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parent / "lab" / "wire-nightly.sh"


def git(repo: Path, *args: str) -> str:
    return subprocess.check_output(
        ["git", "-C", str(repo), *args], stderr=subprocess.DEVNULL, text=True
    ).strip()


class NightlyTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.origin = self.root / "origin"
        self.origin.mkdir()
        git(self.origin, "init", "-b", "main")
        git(self.origin, "config", "user.name", "Nightly test")
        git(self.origin, "config", "user.email", "nightly@example.invalid")
        self.commit("old")
        self.repo = self.root / "repo"
        git(self.root, "clone", str(self.origin), str(self.repo))
        self.expected = self.commit("new")
        self.bin = self.root / "bin"
        self.bin.mkdir()
        go = self.bin / "go"
        go.write_text('#!/bin/sh\nprintf ran >> "$GO_CALLS"\nexit "${GO_STATUS:-0}"\n')
        go.chmod(0o755)

    def commit(self, value):
        (self.origin / "input").write_text(value)
        git(self.origin, "add", "input")
        git(self.origin, "commit", "-qm", value)
        return git(self.origin, "rev-parse", "HEAD")

    def run_nightly(self, **extra):
        env = dict(os.environ, PATH=f"{self.bin}:{os.environ['PATH']}",
                   NIAC_WIRE_REPO=str(self.repo), NIAC_WIRE_STATE=str(self.root / "state"),
                   NIAC_WIRE_TOKEN=str(self.root / "no-token"), NIAC_WALK_CORPUS="",
                   GO_CALLS=str(self.root / "go-calls"), **extra)
        result = subprocess.run(["bash", str(SCRIPT)], env=env, capture_output=True, text=True)
        record = json.loads((self.root / "state" / "last-run.json").read_text())
        return result, record

    def assert_setup_failed(self):
        result, record = self.run_nightly()
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(record["result"], "failed")
        self.assertFalse((self.root / "go-calls").exists())

    def test_clean_checkout_tests_fetched_commit(self):
        result, record = self.run_nightly()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(record["commit"], self.expected)
        self.assertEqual(record["result"], "passed")
        self.assertTrue((self.root / "go-calls").exists())

    def test_dirty_checkout_is_preserved_and_never_tested(self):
        (self.repo / "input").write_text("operator edits")
        self.assert_setup_failed()
        self.assertEqual((self.repo / "input").read_text(), "operator edits")

    def test_untracked_code_is_never_tested(self):
        (self.repo / "untracked.go").write_text("package unexpected")
        self.assert_setup_failed()

    def test_fetch_failure_is_not_a_stale_pass(self):
        git(self.repo, "remote", "set-url", "origin", str(self.root / "missing"))
        self.assert_setup_failed()

    def test_test_failure_is_recorded(self):
        result, record = self.run_nightly(GO_STATUS="17")
        self.assertEqual(result.returncode, 17)
        self.assertEqual(record["exitCode"], 17)
        self.assertEqual(record["result"], "failed")

    def test_missing_clone_replaces_previous_pass(self):
        self.run_nightly()
        self.repo.rename(self.root / "preserved-repo")
        result, record = self.run_nightly()
        self.assertEqual(result.returncode, 78)
        self.assertEqual(record["result"], "failed")
        history = (self.root / "state" / "history.jsonl").read_text().splitlines()
        self.assertEqual([json.loads(row)["result"] for row in history], ["passed", "failed"])

    def test_unwritable_history_cannot_pass(self):
        (self.root / "state" / "history.jsonl").mkdir(parents=True)
        result, _ = self.run_nightly()
        self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
