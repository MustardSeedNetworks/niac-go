#!/usr/bin/env python3
"""Self-test for check-release-notes.py: builds a throwaway repository and
proves the gate goes red for a missing entry, green for a clean entry, and
green for a commit type release-please is not configured to render."""

from __future__ import annotations

import importlib.util
import io
import json
import os
import subprocess
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("gate", HERE / "check-release-notes.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)

CONFIG = {
    "changelog-sections": [
        {"type": "feat", "section": "Features"},
        {"type": "fix", "section": "Bug Fixes"},
        {"type": "chore", "section": "Miscellaneous"},
    ]
}

HEADER = """# Changelog

"""


class Repo:
    """A throwaway repository with two tags and a changelog the test writes."""

    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.git("init", "--quiet", "--initial-branch=main")
        self.git("config", "user.email", "gate@example.com")
        self.git("config", "user.name", "Gate Test")
        config = self.root / ".github" / "release-please-config.json"
        config.parent.mkdir(parents=True)
        config.write_text(json.dumps(CONFIG))
        self.commit("chore: seed the tree (#1)")
        self.git("tag", "v0.1.0")

    def git(self, *args: str) -> None:
        subprocess.run(("git", *args), cwd=self.root, check=True, capture_output=True)

    def commit(self, subject: str) -> None:
        marker = self.root / "tree.txt"
        marker.write_text(subject)
        self.git("add", "-A")
        self.git("commit", "--quiet", "-m", subject)

    def changelog(self, body: str, tag: str | None = "v0.2.0") -> None:
        (self.root / "CHANGELOG.md").write_text(HEADER + body)
        self.git("add", "-A")
        self.git("commit", "--quiet", "-m", "chore(main): release 0.2.0")
        if tag:
            self.git("tag", tag)

    def merge(self, subject: str) -> None:
        """A commit that lands after the release commit, as the queue does."""
        self.commit(subject)

    def run(self, *args: str) -> tuple[int, str]:
        previous = Path.cwd()
        os.chdir(self.root)
        try:
            captured = io.StringIO()
            with redirect_stdout(captured):
                code = gate.main(list(args))
            return code, captured.getvalue()
        finally:
            os.chdir(previous)

    def close(self) -> None:
        self.tmp.cleanup()


class CheckReleaseNotes(unittest.TestCase):
    def setUp(self) -> None:
        self.repo = Repo()
        self.addCleanup(self.repo.close)

    def test_missing_entry_is_red(self) -> None:
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.changelog("## [0.2.0] (2026-01-01)\n\n### Features\n\n* something else (#3)\n")
        code, output = self.repo.run()
        self.assertEqual(code, 1, output)
        self.assertIn("#2", output)

    def test_mentioned_entry_is_green(self) -> None:
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n"
        )
        code, output = self.repo.run()
        self.assertEqual(code, 0, output)

    def test_unrendered_type_is_green(self) -> None:
        """release-please omits a type its config does not name, so requiring
        an entry for one fails a release that is in fact correct."""
        self.repo.commit("build: change how the tree is packaged (#2)")
        self.repo.changelog("## [0.2.0] (2026-01-01)\n\n### Features\n\n* something else (#3)\n")
        code, output = self.repo.run()
        self.assertEqual(code, 0, output)

    def test_breaking_change_is_required_whatever_its_type(self) -> None:
        self.repo.commit("build!: drop a packaging format (#2)")
        self.repo.changelog("## [0.2.0] (2026-01-01)\n\n### Features\n\n* something else (#3)\n")
        code, output = self.repo.run()
        self.assertEqual(code, 1, output)
        self.assertIn("#2", output)


class PendingRelease(unittest.TestCase):
    """--pending checks the release before it is tagged.

    The post-tag check cannot stop the race it reports: by the time it runs,
    the tag exists and the changelog is permanently short. --pending runs on
    the merge-queue tree, where a swept-in commit can still eject the PR.
    """

    def setUp(self) -> None:
        self.repo = Repo()
        self.addCleanup(self.repo.close)

    def test_untagged_release_missing_a_swept_in_commit_is_red(self) -> None:
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n",
            tag=None,
        )
        self.repo.merge("feat: land while the release PR was open (#4)")
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 1, output)
        self.assertIn("#4", output)

    def test_untagged_release_that_describes_the_tree_is_green(self) -> None:
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n",
            tag=None,
        )
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 0, output)

    def test_no_pending_release_is_green(self) -> None:
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n"
        )
        self.repo.merge("feat: ordinary work after the release (#4)")
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 0, output)


if __name__ == "__main__":
    unittest.main()
