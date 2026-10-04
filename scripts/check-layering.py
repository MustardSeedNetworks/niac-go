#!/usr/bin/env python3
"""Fail when a domain package imports an outer layer (niac-go#2435).

docs/ARCHITECTURE.md draws the dependency direction: the simulation and domain
packages under internal/ point inward, and the outer layers compose them. The
outer layers are the HTTPS API (internal/api and its subpackages), the daemon
that wires the API to a running simulation, the CLI (cmd/, tools/ and the
daemon client internal/cliclient) and the test harnesses that drive a built
binary (internal/acceptance, internal/wiretest). A domain file that imports
one of those couples the simulator to its own transport.

Every .go file under internal/ outside those layers is checked, tests
included. Imports are read from the source text rather than `go list`, so
platform-tagged files are checked on every host. CI lints only linux and
windows, so depguard never sees a darwin file.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

MODULE = "github.com/MustardSeedNetworks/niac-go/"
OUTER = (
    "cmd",
    "tools",
    "internal/api",
    "internal/daemon",
    "internal/cliclient",
    "internal/acceptance",
    "internal/wiretest",
)

# Comments and string literals, in one alternation so a `//` inside a string
# is not taken for a comment. Only comments are dropped.
TOKENS = re.compile(r'//[^\n]*|/\*.*?\*/|"(?:\\.|[^"\\\n])*"|`[^`]*`', re.S)
FIRST_DECL = re.compile(r"^(?:func|type|var|const)\b", re.M)
IMPORT = re.compile(r"\bimport\s*(\((?P<group>[^)]*)\)|(?:[\w.]+\s+)?(?P<single>\"[^\"]*\"|`[^`]*`))")
PATH = re.compile(r"\"([^\"]*)\"|`([^`]*)`")


def under(path: str, prefix: str) -> bool:
    return path == prefix or path.startswith(prefix + "/")


def outer_layer(package: str) -> str | None:
    return next((layer for layer in OUTER if under(package, layer)), None)


def imports(source: str) -> list[str]:
    """Import paths of a Go file. Imports precede every other declaration."""
    code = TOKENS.sub(lambda m: m.group(0) if m.group(0)[0] in "\"`" else " ", source)
    header = FIRST_DECL.split(code, maxsplit=1)[0]
    paths: list[str] = []
    for decl in IMPORT.finditer(header):
        specs = decl.group("group") if decl.group("group") is not None else decl.group("single")
        paths.extend(a or b for a, b in PATH.findall(specs))
    return paths


def violations(root: Path) -> set[tuple[str, str]]:
    found: set[tuple[str, str]] = set()
    for file in sorted((root / "internal").rglob("*.go")):
        rel = file.relative_to(root).as_posix()
        if "/testdata/" in rel or outer_layer(file.parent.relative_to(root).as_posix()):
            continue
        for path in imports(file.read_text(encoding="utf-8")):
            if path.startswith(MODULE) and outer_layer(path.removeprefix(MODULE)):
                found.add((rel, path.removeprefix(MODULE)))
    return found


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    root = parser.parse_args().root

    found = sorted(violations(root))
    for rel, path in found:
        print(f"FAIL: {rel} imports {path}, an outer layer (see docs/ARCHITECTURE.md)")
    if found:
        print("\nMove the dependency inward: the outer layer maps domain types, not the reverse.")
        return 1

    print("OK: no domain package imports an outer layer")
    return 0


if __name__ == "__main__":
    sys.exit(main())
