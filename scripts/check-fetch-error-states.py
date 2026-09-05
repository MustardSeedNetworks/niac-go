#!/usr/bin/env python3
"""check-fetch-error-states.py — every fetch must be able to fail visibly.

`useApiResource` returns `{data, loading, error, refetch}` and can raise a
toast via its `errorToast` option. Six of eight call sites destructured only
`data` and `loading`, so a failed fetch rendered as an empty page: the topology
looked like a network with no devices, the device list like a daemon with none.

TopologyPage's own comment claimed "errors surface through the per-hook error
states" while never reading them, which is worse than no comment.

A call site is SATISFIED when it either destructures `error` or passes
`errorToast`. This is a ratchet against scripts/fetch-error-states-baseline.txt:
growth fails, and a baselined file that has since been fixed fails too, so the
baseline stays a work queue rather than an allow-list.

Run locally: scripts/check-fetch-error-states.py [--update]
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

UI = Path("ui/src")
BASELINE = Path("scripts/fetch-error-states-baseline.txt")

# Match the destructuring assignment that consumes the hook, not the call:
# which fields the page took is the whole question.
CALL = re.compile(
    r"(?:const|let)\s*(?P<binding>\{.*?\}|\w+)\s*=\s*useApiResource\((?P<args>.*?)\);",
    re.DOTALL,
)


def unsatisfied(text: str) -> list[str]:
    """Return the line of each call site that cannot show a failure."""
    found = []
    for match in CALL.finditer(text):
        binding, args = match.group("binding"), match.group("args")
        if re.search(r"\berror\b", binding) or "errorToast" in args:
            continue
        found.append(str(text[: match.start()].count("\n") + 1))
    return found


def scan() -> dict[str, list[str]]:
    results: dict[str, list[str]] = {}
    for path in sorted(UI.rglob("*.tsx")) + sorted(UI.rglob("*.ts")):
        if path.name.endswith((".test.tsx", ".test.ts", ".stories.tsx")):
            continue
        text = path.read_text(encoding="utf-8")
        if "useApiResource(" not in text:
            continue
        lines = unsatisfied(text)
        if lines:
            results[str(path)] = lines
    return results


def load_baseline() -> set[str]:
    if not BASELINE.exists():
        return set()
    return {
        line.strip()
        for line in BASELINE.read_text(encoding="utf-8").splitlines()
        if line.strip() and not line.startswith("#")
    }


def main() -> int:
    results = scan()
    current = set(results)

    if "--update" in sys.argv:
        BASELINE.write_text(
            "# Files with a useApiResource call that cannot show a failure.\n"
            "# Shrink-only: see scripts/check-fetch-error-states.py\n"
            + "".join(f"{name}\n" for name in sorted(current)),
            encoding="utf-8",
        )
        print(f"baseline written: {len(current)} file(s)")
        return 0

    baseline = load_baseline()
    added = sorted(current - baseline)
    fixed = sorted(baseline - current)

    for name in added:
        print(
            f"::error::{name} has a useApiResource that cannot show a failure "
            f"(line(s) {', '.join(results[name])})"
        )
    for name in fixed:
        print(
            f"::error::{name} no longer needs to be baselined — remove it "
            f"(run scripts/check-fetch-error-states.py --update)"
        )

    if added or fixed:
        print(
            "\nA fetch that cannot fail visibly renders an empty page instead of an\n"
            "error: destructure `error` and render it, or pass `errorToast` for a\n"
            "background poll."
        )
        return 1

    print(
        "Fetch-error-state gate: every useApiResource call site can show a failure "
        f"({len(baseline)} baselined)."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
