#!/usr/bin/env python3
"""check-docs-truth.py — operator docs describe things that exist.

The 2026-09-15 fleet defect sweep (#2180) found three ways `docs/` had drifted
away from the binary it documents, all of them invisible to CI: environment
variables the Go source has never read, an invocation the root command rejects,
`examples/...` paths handed out without saying the directory ships empty, and
links to files that are not in the repository. The weekly Docs Link Check
walks external URLs, so it is slow, schedule-only, and blind to the first
three.

Three offline checks, each deterministic and fast enough to run per-PR:

  A. Every environment variable named in `docs/CLI_REFERENCE.md`'s
     "Environment Variables" section is actually read by the binary --
     `os.Getenv`/`LookupEnv` on that literal name. Matching the bare string
     anywhere in the source would pass a name that only appears in a comment
     or a help string, which is how `NIAC_API_TOKEN` reads as wired from three
     files that merely mention it.
  B. Every relative markdown link under `docs/` resolves. `docs/archive/` is
     excluded: it is a frozen record of what was written at the time, and
     rewriting its links would falsify it rather than fix anything.
  C. A doc that hands out a repository `examples/` path also names
     `scripts/sync-demo-catalog.sh`, because `examples/` ships holding only
     its README and is populated from the private demo catalog. Markdown link
     targets are check B's, not this one's -- `docs/examples/` is a different,
     tracked directory, and reading a link target as a catalog path is how the
     first draft of this check flagged it. A bare `../examples/` link is still
     counted here: the directory resolves, so check B is happy, while
     "Ready-to-use configurations" points at a lone README.

Run locally: scripts/check-docs-truth.py
"""

from __future__ import annotations

import os
import re
import subprocess
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

CLI_REFERENCE = "docs/CLI_REFERENCE.md"
SYNC_SCRIPT = "scripts/sync-demo-catalog.sh"

# docs/archive/ is a historical record; see the module docstring.
EXCLUDED_DIRS = ("docs/archive/",)

# Check C exempts the document whose subject IS the catalog: it explains the
# sync rather than assuming it.
CATALOG_DOCS = ("docs/SHARED_DEMO_CATALOG.md",)

# Paths under examples/ that are not the operator reading the catalog, so the
# sync prerequisite would be the wrong sentence to add.
NOT_CATALOG_READS = {
    # sanitize writes here; the operator creates the directory.
    ("docs/WALK_FILE_SANITIZATION.md", "examples/device_walks/"),
    # an editor's yaml.schemas glob, not a path handed to the reader.
    ("docs/schemas/README.md", "examples/"),
}

ENV_SECTION = re.compile(
    r"^## Environment Variables$(.*?)(?=^## )", re.MULTILINE | re.DOTALL
)
ENV_VAR = re.compile(r"\b([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)\b")
MD_LINK = re.compile(r"\[[^\]]*\]\(([^)]+)\)")
EXAMPLES_PATH = re.compile(r"(?<![\w/])(?:\.\./)?examples/[\w./-]*")


def docs_files() -> list[str]:
    out = subprocess.run(
        ["git", "ls-files", "docs/*.md", "docs/**/*.md"],
        cwd=REPO,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.split()
    return [f for f in out if not f.startswith(EXCLUDED_DIRS)]


def go_sources() -> str:
    files = subprocess.run(
        ["git", "ls-files", "*.go"],
        cwd=REPO,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.split()
    parts = []
    for f in files:
        with open(os.path.join(REPO, f), encoding="utf-8") as fh:
            parts.append(fh.read())
    return "\n".join(parts)


def check_env_vars(failures: list[str]) -> None:
    with open(os.path.join(REPO, CLI_REFERENCE), encoding="utf-8") as fh:
        text = fh.read()
    section = ENV_SECTION.search(text)
    if not section:
        failures.append(f"{CLI_REFERENCE}: no '## Environment Variables' section")
        return
    offset = text[: section.start(1)].count("\n") + 1

    go = go_sources()
    seen: set[str] = set()
    for line_no, line in enumerate(section.group(1).splitlines(), offset + 1):
        for name in ENV_VAR.findall(line):
            if name in seen:
                continue
            seen.add(name)
            read = re.search(
                r"os\.(?:Getenv|LookupEnv)\(\s*\"" + re.escape(name) + r"\"",
                go,
            ) or re.search(
                r"=\s*\"" + re.escape(name) + r"\"",
                go,
            )
            if read:
                continue
            failures.append(
                f"{CLI_REFERENCE}:{line_no}: documents {name}, which the binary "
                f"never reads (no os.Getenv/LookupEnv on that name)"
            )


def check_links(failures: list[str]) -> None:
    for f in docs_files():
        with open(os.path.join(REPO, f), encoding="utf-8") as fh:
            text = fh.read()
        for line_no, line in enumerate(text.splitlines(), 1):
            for target in MD_LINK.findall(line):
                target = target.split("#")[0].strip()
                if not target or target.startswith(
                    ("http://", "https://", "mailto:", "#")
                ):
                    continue
                resolved = os.path.normpath(
                    os.path.join(REPO, os.path.dirname(f), target)
                )
                if not os.path.exists(resolved):
                    failures.append(f"{f}:{line_no}: link {target} does not resolve")


def check_examples_prerequisite(failures: list[str]) -> None:
    for f in docs_files():
        if f in CATALOG_DOCS:
            continue
        with open(os.path.join(REPO, f), encoding="utf-8") as fh:
            text = fh.read()
        if SYNC_SCRIPT in text:
            continue
        link_targets = set(MD_LINK.findall(text))
        hits = [
            h
            for h in EXAMPLES_PATH.findall(text)
            if h not in link_targets and (f, h) not in NOT_CATALOG_READS
        ]
        if not hits:
            continue
        failures.append(
            f"{f}: hands out {hits[0]} but never names {SYNC_SCRIPT}; "
            f"examples/ ships holding only its README"
        )


def main() -> int:
    failures: list[str] = []
    check_env_vars(failures)
    check_links(failures)
    check_examples_prerequisite(failures)

    if failures:
        print("docs truth check FAILED:\n", file=sys.stderr)
        for f in failures:
            print(f"  {f}", file=sys.stderr)
        print(f"\n{len(failures)} problem(s).", file=sys.stderr)
        return 1

    print("✓ docs truth: env vars, relative links and examples/ prerequisites OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
