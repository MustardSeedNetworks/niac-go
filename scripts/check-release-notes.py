#!/usr/bin/env python3
"""check-release-notes.py — a tag's contents must match its changelog entry.

v0.95.1 shipped four commits and its CHANGELOG section listed one, because the
release PR auto-merged before release-please could regenerate it: the PR's own
checks go green faster than CI on main completes for a commit merged moments
earlier, and release-please only runs after that CI succeeds (niac#1817).

Nothing self-corrects afterwards. The next release compares against the new
tag, so a commit missed here is missing from the changelog permanently.

Every commit between the previous tag and this one must be mentioned in this
version's CHANGELOG section, matched on its PR number, which release-please
always writes.

The post-tag form reports the loss but cannot prevent it. `--pending` runs
the same comparison against the working tree before release-please's PR is
merged, so a commit the merge queue swept in ejects the release PR instead of
shipping inside its tag.

Usage:
  scripts/check-release-notes.py               # newest tag
  scripts/check-release-notes.py v0.95.1       # a specific tag
  scripts/check-release-notes.py --pending     # an untagged release in the tree
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

CHANGELOG = Path("CHANGELOG.md")
RELEASE_PLEASE_CONFIG = Path(".github/release-please-config.json")

# release-please's own commits describe the release, not its contents.
RELEASE_COMMIT = re.compile(r"^chore\(main\): release ")
# A conventional commit is the unit a changelog entry is generated from.
CONVENTIONAL = re.compile(r"^([a-z]+)(?:\([^)]*\))?(!)?: ")
# GitHub appends "(#123)" to a squashed merge; release-please links the same id.
PR_NUMBER = re.compile(r"\(#(\d+)\)\s*$")


def rendered_types() -> set[str]:
    """The commit types release-please writes into the changelog.

    A type the config does not name is dropped from the generated notes on
    purpose, so demanding an entry for it fails a release that is correct --
    v0.95.40's `build:` commit was the first to hit that.
    """
    config = json.loads(RELEASE_PLEASE_CONFIG.read_text(encoding="utf-8"))
    return {section["type"] for section in config["changelog-sections"]}


def run(*args: str) -> str:
    return subprocess.run(args, capture_output=True, text=True, check=True).stdout.strip()


def tags() -> list[str]:
    return run("git", "tag", "--sort=-v:refname").splitlines()


def commits_in(previous: str, tag: str, rendered: set[str]) -> list[tuple[str, str]]:
    """Return (pr_number, subject) for each content commit in the range."""
    span = f"{previous}..{tag}" if previous else tag
    found = []
    for line in run("git", "log", "--format=%s", span).splitlines():
        if RELEASE_COMMIT.match(line):
            continue
        conventional = CONVENTIONAL.match(line)
        if not conventional:
            continue
        commit_type, breaking = conventional.groups()
        # A breaking change is always announced, whatever its type.
        if commit_type not in rendered and not breaking:
            continue
        match = PR_NUMBER.search(line)
        found.append((match.group(1) if match else "", line))
    return found


def section_for(version: str, text: str) -> str | None:
    """Return the CHANGELOG body for one version, or None when it has none."""
    # release-please writes "## [0.95.1](compare-url) (date)" per release.
    starts = [m for m in re.finditer(r"^## \[?([0-9]+\.[0-9]+\.[0-9]+)\]?", text, re.MULTILINE)]
    for index, match in enumerate(starts):
        if match.group(1) != version:
            continue
        end = starts[index + 1].start() if index + 1 < len(starts) else len(text)
        return text[match.start() : end]
    return None


def newest_version(text: str) -> str | None:
    match = re.search(r"^## \[?([0-9]+\.[0-9]+\.[0-9]+)\]?", text, re.MULTILINE)
    return match.group(1) if match else None


def check_pending(known: list[str]) -> int:
    """Check a release the tree describes but no tag names yet."""
    text = CHANGELOG.read_text(encoding="utf-8")
    version = newest_version(text)
    if version is None:
        print(f"No release section in {CHANGELOG}; nothing pending.")
        return 0
    if f"v{version}" in known:
        print(f"v{version} is already tagged; no pending release to check.")
        return 0

    section = section_for(version, text)
    assert section is not None
    rendered = rendered_types()
    missing = [
        subject
        for number, subject in commits_in(known[0] if known else "", "HEAD", rendered)
        if not number or f"#{number}" not in section
    ]
    if missing:
        print(f"::error::the pending v{version} release would ship commits its entry omits:")
        for subject in missing:
            print(f"  {subject}")
        print(
            "\nA commit landed on main while the release PR was open, so the PR no\n"
            "longer describes what its tag would contain (niac#1817). Let\n"
            "release-please regenerate the PR instead of merging this one."
        )
        return 1

    print(f"pending v{version}: every commit appears in the changelog entry.")
    return 0


def main(argv: list[str] | None = None) -> int:
    args = sys.argv[1:] if argv is None else argv
    known = tags()
    if args and args[0] == "--pending":
        return check_pending(known)

    tag = args[0] if args else None
    if not known:
        print("No tags yet; nothing to check.")
        return 0
    if tag is None:
        tag = known[0]
    if tag not in known:
        print(f"::error::unknown tag {tag}")
        return 1

    previous = known[known.index(tag) + 1] if known.index(tag) + 1 < len(known) else ""
    version = tag.lstrip("v")

    section = section_for(version, CHANGELOG.read_text(encoding="utf-8"))
    if section is None:
        print(f"::error::{tag} has no section in {CHANGELOG}")
        return 1

    rendered = rendered_types()
    missing = [
        subject
        for number, subject in commits_in(previous, tag, rendered)
        if not number or f"#{number}" not in section
    ]
    if missing:
        print(f"::error::{tag} ships commits its changelog entry does not mention:")
        for subject in missing:
            print(f"  {subject}")
        print(
            "\nThe release PR merged before release-please regenerated it (niac#1817).\n"
            "Backfill this version's CHANGELOG section by hand; the next release\n"
            "compares against this tag, so nothing else will pick these up."
        )
        return 1

    counted = len(commits_in(previous, tag, rendered))
    print(f"{tag}: all {counted} commit(s) appear in the changelog entry.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
