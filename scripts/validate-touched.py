#!/usr/bin/env python3
"""validate-touched.py — validate what the branch touched, not the whole tree.

The inner loop. `make test` runs every Go package under -race, the whole Vitest
suite and every hook test; a branch that edits one leaf package needs that
package, everything in the module that imports it (directly or through another
package, and through tests), and the gates that read the files it changed.
`make test` still runs once before the PR; this is what runs between edits.

The touched set is every path that differs between the merge base with $BASE
(default origin/main) and the working tree, committed or not, plus untracked
files. Renames count as a delete and an add, so the package a file left is
validated too.

Every command is printed before it runs, and a failing step does not stop the
others: the exit status is non-zero when any step failed.

Run locally: make validate-touched   (or scripts/validate-touched.py --dry-run)
"""

from __future__ import annotations

import argparse
import fnmatch
import os
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parent.parent

# A change to any of these can alter how every package builds or lints.
WHOLE_MODULE = ("go.mod", "go.sum", ".golangci.yml")

# A change to any of these can alter how every UI test resolves or runs.
WHOLE_UI = (
    "ui/package.json",
    "ui/package-lock.json",
    "ui/vite.config.ts",
    "ui/vitest.config.ts",
    "ui/tsconfig*.json",
    "ui/src/test/*",
)

UI_SOURCE_SUFFIXES = {".ts", ".tsx", ".css", ".json"}

# The same entry point ui/package.json's `test` script uses.
VITEST = ["node", "--disable-warning=DEP0205", "./node_modules/vitest/vitest.mjs"]

# Locale catalogues live with the Go i18n package but the UI imports them too.
UI_INPUTS = ("ui/src/*", "internal/i18n/locales/*")


def matches_any(path: str, patterns: tuple[str, ...]) -> bool:
    return any(fnmatch.fnmatchcase(path, p) for p in patterns)


@dataclass(frozen=True)
class Gate:
    """A scripts/check-* gate and the paths it reads.

    Patterns are fnmatch patterns over repo-relative paths, where `*` also
    crosses `/`. The gate's own files (the script, its self-test, its baseline
    or allow-list: `scripts/<name>-*`) are inputs of every gate implicitly.
    """

    script: str
    inputs: tuple[str, ...]

    @property
    def name(self) -> str:
        return PurePosixPath(self.script).stem.removeprefix("check-")

    def own_files(self) -> tuple[str, ...]:
        return (
            f"scripts/{PurePosixPath(self.script).name}",
            f"scripts/test-check-{self.name}.*",
            f"scripts/{self.name}-*",
        )

    def reads(self, path: str) -> bool:
        return matches_any(path, self.inputs + self.own_files())

    def commands(self) -> list[list[str]]:
        cmds = []
        self_test = ROOT / "scripts" / f"test-check-{self.name}.py"
        if self_test.exists():
            cmds.append(["python3", f"scripts/{self_test.name}"])
        if self.script.endswith(".py"):
            cmds.append(["python3", self.script])
        else:
            cmds.append([f"./{self.script}"])
        return cmds


GATES = (
    Gate(
        "scripts/check-authoring-parity.py",
        ("docs/schemas/*", "ui/src/components/*"),
    ),
    Gate("scripts/check-banned-vocabulary.py", ("*",)),
    Gate("scripts/check-component-variants.py", ("ui/src/*",)),
    Gate("scripts/check-docs-truth.py", ("docs/*", "examples/*", "*.go")),
    Gate("scripts/check-fetch-error-states.py", ("ui/src/*",)),
    Gate("scripts/check-file-size.sh", ("*.go", "ui/src/*")),
    Gate("scripts/check-filename-policy.sh", ("*.go",)),
    Gate(
        "scripts/check-https-only.sh",
        (
            "cmd/niac/*",
            "internal/api/*",
            "internal/daemon/*",
            "deploy/macos/build-pkg.sh",
            "deploy/windows/build.ps1",
            "docs/*.md",
            "docs/openapi.yaml",
            "tests/smoke/run_smoke_tests.sh",
            "ui/vite.config.ts",
        ),
    ),
    Gate("scripts/check-json-casing.sh", ("internal/api/*",)),
    Gate("scripts/check-library-stdout.sh", ("internal/*.go",)),
    Gate("scripts/check-output-escaping.sh", ("internal/api/*", "ui/src/*")),
    Gate("scripts/check-package-reachability.sh", ("*.go", "go.mod")),
    Gate(
        "scripts/check-page-stories.py",
        ("ui/src/pageRegistry.ts", "ui/src/pages/*", "ui/src/test/storybook/*"),
    ),
    Gate(
        "scripts/check-route-consumers.py",
        ("internal/api/*", "internal/cliclient/*", "cmd/*", "ui/src/*"),
    ),
    Gate("scripts/check-starter-walks.sh", ("internal/library/starter/walks/*",)),
    Gate("scripts/check-token-discipline.sh", ("ui/src/*",)),
    Gate("scripts/check-tsconfig-flags.py", ("ui/tsconfig*.json",)),
)

# Gates with no file inputs to select on. check-stale-tests is a precondition
# the make target runs first; check-release-notes compares CHANGELOG.md with
# the commits since a tag, which only means something on the release branch.
NOT_INPUT_DRIVEN = {"scripts/check-stale-tests.sh", "scripts/check-release-notes.py"}


@dataclass(frozen=True)
class GoPackage:
    import_path: str
    rel_dir: str
    imports: frozenset[str]
    test_imports: frozenset[str]


def run_capture(cmd: list[str]) -> str:
    return subprocess.run(
        cmd, cwd=ROOT, check=True, capture_output=True, text=True
    ).stdout


def touched_files(base: str) -> list[str]:
    merge_base = run_capture(["git", "merge-base", base, "HEAD"]).strip()
    diff = run_capture(["git", "diff", "--name-only", "--no-renames", merge_base])
    untracked = run_capture(["git", "ls-files", "--others", "--exclude-standard"])
    return sorted(set(diff.split()) | set(untracked.split()))


def go_packages() -> dict[str, GoPackage]:
    """Every package in the module under the host's default build tags, by import path."""
    sep = "\x1f"
    fmt = sep.join(
        [
            "{{.ImportPath}}",
            "{{.Dir}}",
            '{{join .Imports ","}}',
            '{{join .TestImports ","}}',
            '{{join .XTestImports ","}}',
        ]
    )
    pkgs = {}
    for line in run_capture(["go", "list", "-f", fmt, "./..."]).splitlines():
        path, directory, imports, test_imports, xtest_imports = line.split(sep)
        pkgs[path] = GoPackage(
            import_path=path,
            rel_dir=Path(directory).relative_to(ROOT).as_posix(),
            imports=frozenset(filter(None, imports.split(","))),
            test_imports=frozenset(
                filter(None, (test_imports + "," + xtest_imports).split(","))
            ),
        )
    return pkgs


def owning_package(path: str, by_dir: dict[str, str]) -> str | None:
    """The package a changed file belongs to, if any.

    A .go file belongs to the package in its own directory. Any other file —
    an embedded asset, anything under testdata, including .go fixtures there —
    belongs to the nearest enclosing package, because its tests read it.
    """
    directory = PurePosixPath(path).parent
    if path.endswith(".go") and "testdata" not in directory.parts:
        return by_dir.get(directory.as_posix())
    while directory.as_posix() not in by_dir:
        if directory == PurePosixPath("."):
            return None
        directory = directory.parent
    return by_dir[directory.as_posix()]


def reverse_dependencies(changed: set[str], pkgs: dict[str, GoPackage]) -> set[str]:
    """Changed packages, every in-module package that imports one of them
    directly or transitively, and every package whose tests import any of those."""
    importers: dict[str, set[str]] = {p: set() for p in pkgs}
    for pkg in pkgs.values():
        for dep in pkg.imports & pkgs.keys():
            importers[dep].add(pkg.import_path)
    affected = set(changed)
    frontier = list(changed)
    while frontier:
        for importer in importers[frontier.pop()]:
            if importer not in affected:
                affected.add(importer)
                frontier.append(importer)
    affected |= {p.import_path for p in pkgs.values() if p.test_imports & affected}
    return affected


@dataclass(frozen=True)
class Step:
    label: str
    cmd: list[str]
    cwd: str = "."


@dataclass
class Plan:
    steps: list[Step]
    notes: list[str]


def plan_go(
    touched: list[str], pkgs: dict[str, GoPackage], plan: Plan, golangci: str
) -> None:
    whole = [f for f in touched if f in WHOLE_MODULE]
    if whole:
        plan.notes.append(
            f"Go: {', '.join(whole)} changed, so every package is in scope"
        )
        plan.steps.append(
            Step("golangci-lint", [golangci, "run", "--timeout=5m", "./..."])
        )
        plan.steps.append(Step("go test", ["go", "test", "-race", "./..."]))
        return
    by_dir = {p.rel_dir: p.import_path for p in pkgs.values()}
    changed = set()
    for path in touched:
        owner = owning_package(path, by_dir)
        if owner:
            changed.add(owner)
        elif path.endswith(".go"):
            plan.notes.append(
                f"Go: {path} is in no package under this host's build tags"
            )
    if not changed:
        plan.notes.append("Go: no package touched, no Go lint or tests")
        return
    affected = reverse_dependencies(changed, pkgs)
    plan.notes.append(
        f"Go: {len(changed)} package(s) touched, {len(affected) - len(changed)} reverse dependent(s)"
    )
    lint_dirs = sorted(f"./{pkgs[p].rel_dir}" for p in changed)
    plan.steps.append(
        Step("golangci-lint", [golangci, "run", "--timeout=5m", *lint_dirs])
    )
    plan.steps.append(Step("go test", ["go", "test", "-race", *sorted(affected)]))


def plan_ui(touched: list[str], plan: Plan) -> None:
    whole = [f for f in touched if matches_any(f, WHOLE_UI)]
    sources = [
        f
        for f in touched
        if matches_any(f, UI_INPUTS) and PurePosixPath(f).suffix in UI_SOURCE_SUFFIXES
    ]
    deleted = [f for f in sources if not (ROOT / f).exists()]
    present = [os.path.relpath(f, "ui") for f in sources if f not in deleted]
    if whole or deleted:
        reason = whole or [f"{f} (deleted)" for f in deleted]
        plan.notes.append(
            f"UI: {', '.join(reason)} changed, so the whole Vitest suite runs"
        )
        plan.steps.append(Step("vitest", ["npm", "test"], cwd="ui"))
    elif present:
        plan.steps.append(
            Step("vitest", [*VITEST, "related", "--run", *present], cwd="ui")
        )
    else:
        plan.notes.append("UI: no UI source touched, no Vitest")
    biome = [f for f in present if f.startswith("src/")]
    if biome:
        plan.steps.append(
            Step("biome", ["npx", "@biomejs/biome", "check", *biome], cwd="ui")
        )


def plan_docs(touched: list[str], plan: Plan, markdownlint: str) -> None:
    docs = [f for f in touched if f.endswith(".md") and (ROOT / f).exists()]
    if docs:
        cmd = ["npx", "--yes", f"markdownlint-cli2@{markdownlint}", *docs]
        plan.steps.append(Step("markdownlint", cmd))


def plan_gates(touched: list[str], plan: Plan) -> None:
    for gate in GATES:
        if any(gate.reads(f) for f in touched):
            for cmd in gate.commands():
                plan.steps.append(Step(f"gate {gate.name}", cmd))


def build_plan(
    touched: list[str], pkgs: dict[str, GoPackage], golangci: str, markdownlint: str
) -> Plan:
    plan = Plan(steps=[], notes=[])
    plan_go(touched, pkgs, plan, golangci)
    plan_ui(touched, plan)
    plan_docs(touched, plan, markdownlint)
    plan_gates(touched, plan)
    return plan


def lint_pin(name: str) -> str:
    """A tool version pinned in mk/lint.mk, which in turn matches ci.yml."""
    for line in (ROOT / "mk" / "lint.mk").read_text().splitlines():
        key, _, value = line.partition(":=")
        if key.strip() == name:
            return value.strip()
    sys.exit(f"validate-touched: mk/lint.mk pins no {name}")


def golangci_binary() -> str:
    """The GOPATH golangci-lint, refused unless it is the version CI pins."""
    want = lint_pin("GOLANGCI_LINT_VERSION")
    binary = str(
        Path(run_capture(["go", "env", "GOPATH"]).strip()) / "bin" / "golangci-lint"
    )
    try:
        version = run_capture([binary, "version"])
    except (OSError, subprocess.CalledProcessError):
        version = ""
    if f"version {want.removeprefix('v')} " not in version:
        sys.exit(
            f"validate-touched: {binary} is not golangci-lint {want}; "
            "`make lint-backend` installs the pinned version"
        )
    return binary


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--base", default=os.environ.get("BASE", "origin/main"))
    parser.add_argument(
        "--dry-run", action="store_true", help="print the plan, run nothing"
    )
    args = parser.parse_args()

    touched = touched_files(args.base)
    print(f"validate-touched: {len(touched)} path(s) differ from {args.base}")
    if not touched:
        return 0
    golangci = "golangci-lint" if args.dry_run else golangci_binary()
    plan = build_plan(
        touched, go_packages(), golangci, lint_pin("MARKDOWNLINT_CLI2_VERSION")
    )
    for note in plan.notes:
        print(f"  {note}")

    failed = []
    for step in plan.steps:
        where = "" if step.cwd == "." else f"(cd {step.cwd}) "
        print(f"\n+ {where}{' '.join(step.cmd)}", flush=True)
        if args.dry_run:
            continue
        started = time.monotonic()
        status = subprocess.run(step.cmd, cwd=ROOT / step.cwd, check=False).returncode
        print(f"  [{step.label}: exit {status}, {time.monotonic() - started:.1f} s]")
        if status != 0:
            failed.append(step.label)
    if failed:
        print(f"\nvalidate-touched: FAILED: {', '.join(failed)}")
        return 1
    verdict = "planned" if args.dry_run else "passed"
    print(f"\nvalidate-touched: {len(plan.steps)} step(s) {verdict}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
