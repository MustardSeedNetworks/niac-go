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
        self.git("commit", "--quiet", "-m", f"chore(main): release {gate.newest_version(body)}")
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

    def test_untagged_release_missing_a_pr_queued_ahead_is_red(self) -> None:
        """The release PR's own merge group: a PR queued ahead of it landed a
        commit its entry never lists, and the release commit sits on top."""
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.commit("feat: queued ahead of the release PR (#4)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n",
            tag=None,
        )
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 1, output)
        self.assertIn("#4", output)

    def test_pr_queued_behind_a_short_release_is_green(self) -> None:
        """G-18: the release PR's group fails on #4; the PR queued behind it
        must not be ejected for the same entry."""
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.commit("feat: queued ahead of the release PR (#4)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n",
            tag=None,
        )
        self.repo.merge("feat: queued behind the release PR (#5)")
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 0, output)
        self.assertIn("is not the v0.2.0 release commit", output)

    def test_untagged_section_without_its_release_commit_is_red(self) -> None:
        (self.repo.root / "CHANGELOG.md").write_text(
            HEADER + "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n"
        )
        self.repo.commit("fix: correct a real defect (#2)")
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 1, output)

    def test_commit_merged_after_an_untagged_release_is_green(self) -> None:
        """niac-go#2628: the release PR merged and the next PR's merge-group run
        started before release-please tagged it. That commit is not in the
        release; it belongs to the next one."""
        self.repo.commit("fix: correct a real defect (#2)")
        self.repo.changelog(
            "## [0.2.0] (2026-01-01)\n\n### Bug Fixes\n\n* correct a real defect (#2)\n",
            tag=None,
        )
        self.repo.merge("feat: queued behind the release PR (#4)")
        code, output = self.repo.run("--pending")
        self.assertEqual(code, 0, output)

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


class ReleasePrEnqueue(unittest.TestCase):
    """The guard ejects a release PR whose merge group holds a commit its entry
    omits; it stays live only if nothing can queue ahead of the release PR.
    Auto-merge enqueued it at the back, and PRs ahead of it ejected it on
    every regeneration (seed#3484, G-18)."""

    WORKFLOW = Path(__file__).resolve().parent.parent / ".github" / "workflows" / "release-please.yml"

    def setUp(self) -> None:
        text = self.WORKFLOW.read_text(encoding="utf-8")
        start = text.index("- name: Enqueue the release PR at the front of the merge queue")
        end = text.find("\n      - name:", start + 1)
        self.text = text
        self.step = text[start : end if end != -1 else len(text)]

    def test_release_pr_jumps_to_the_front_of_the_queue(self) -> None:
        self.assertIn("enqueuePullRequest(input: {pullRequestId: $id, jump: $jump})", self.step)
        # The first enqueue asks for the front. jump is a parameter so the
        # same retry loop can fall back to the back of an empty queue.
        self.assertIn("enqueue_pr true", self.step)
        self.assertIn('-F jump="${jump}"', self.step)
        self.assertNotIn("gh pr merge", self.step)

    def test_release_pr_ci_triggers_the_enqueue(self) -> None:
        self.assertIn("branches: [main, release-please--branches--main]", self.text)
        self.assertIn("gh pr checks", self.step)

    def test_enqueue_waits_for_ci_complete_and_does_not_fail_when_unreported(self) -> None:
        """gh pr checks --required exits 0 before CI Complete exists. The
        queue then rejects with 'is expected', and that must not fail the
        run (five red release-please runs on 2026-10-16)."""
        self.assertIn('select(.name == "CI Complete")', self.step)
        self.assertIn('!= "pass"', self.step)
        self.assertIn("not enqueuing yet.", self.step)
        self.assertIn("is expected", self.step)
        self.assertIn("return 2", self.step)
        # The unreported-check path leaves the job green. A hard refusal
        # still exits 1 after both attempts.
        self.assertIn("Required status check CI Complete has not reported; not enqueuing yet.", self.step)
        self.assertLess(
            self.step.index("is expected"),
            self.step.rindex("exit 1"),
        )

    def test_unreported_ci_complete_does_not_fail_the_enqueue(self) -> None:
        """Run the gate and the enqueue loop with a fake gh.

        The real failure was an exit 1 from the no-jump fallback while CI
        Complete had not reported. Both of those shapes must exit 0.
        """
        start = self.step.index('ci_complete="pending"')
        script = self.step[start:]
        # The workflow sleeps 5+10+15+20s across the retries. The test only
        # cares about the exit status.
        script = script.replace('sleep "${delay}"', ":")
        cases = {
            "missing": ("pending", "is expected", 0),
            "passing-but-queue-lags": ("pass", "is expected", 0),
            "queued": ("pass", "", 0),
            # A refusal that is not "check has not reported" still fails the
            # job once the empty-queue fallback has also been refused.
            "hard-refusal": ("pass", "App may not jump the queue", 1),
        }
        for name, (bucket, refusal, expect) in cases.items():
            with self.subTest(name=name):
                code, output = _run_enqueue_fragment(script, bucket, refusal)
                self.assertEqual(code, expect, output)
                if bucket != "pass":
                    self.assertNotIn("Enqueuing", output)
                if refusal == "is expected":
                    self.assertNotIn("Process completed", output)
                    self.assertIn("not enqueuing yet", output)


def _run_enqueue_fragment(script: str, bucket: str, refusal: str) -> tuple[int, str]:
    checks = "[]" if bucket == "pending" else f'[{{"name":"CI Complete","bucket":"{bucket}"}}]'
    with tempfile.TemporaryDirectory() as tmp:
        fake = Path(tmp) / "gh"
        fake.write_text(
            "\n".join(
                [
                    "#!/bin/bash",
                    'if [[ "$*" == *"--required"* ]]; then exit 0; fi',
                    'if [[ "$*" == *"pr checks"* ]]; then printf "%s\\n" "$CHECKS_JSON"; exit 0; fi',
                    'if [[ "$*" == *"pr view"* ]]; then echo PR_NODE_ID; exit 0; fi',
                    'if [[ "$*" == *"totalCount"* ]]; then echo 0; exit 0; fi',
                    'if [[ "$*" == *graphql* ]]; then',
                    '  if [[ -n "$ENQUEUE_REFUSAL" ]]; then',
                    '    echo "Pull request Required status check \\"CI Complete\\" $ENQUEUE_REFUSAL." >&2',
                    "    exit 1",
                    "  fi",
                    '  echo "queued at position 1"',
                    "  exit 0",
                    "fi",
                    'echo "unexpected gh: $*" >&2',
                    "exit 99",
                    "",
                ]
            ),
            encoding="utf-8",
        )
        fake.chmod(0o755)
        env = os.environ.copy()
        env["PATH"] = f"{tmp}:{env['PATH']}"
        env["CHECKS_JSON"] = checks
        env["ENQUEUE_REFUSAL"] = refusal
        env["GITHUB_REPOSITORY"] = "acme/niac"
        env["GITHUB_REPOSITORY_OWNER"] = "acme"
        env["pr"] = "2744"
        env["ENQUEUE"] = "mutation { enqueuePullRequest }"
        env["QUEUE_DEPTH"] = "query { mergeQueue }"
        completed = subprocess.run(
            ["bash", "-euo", "pipefail", "-c", script],
            env=env,
            capture_output=True,
            text=True,
            check=False,
        )
        return completed.returncode, completed.stdout + completed.stderr


if __name__ == "__main__":
    unittest.main()
