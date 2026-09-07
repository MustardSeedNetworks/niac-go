#!/usr/bin/env python3
"""check-page-stories.py — every route owes an Empty, Loaded and Error story.

Before U6 the Storybook a11y gate covered only atoms: two of sixteen routes
had a story, so a whole page's rendered markup was never put in front of axe.
The two page stories it did have found real defects immediately (U2b), and the
fourteen added since found two more — two unnamed `<select>` filters and an
unnamed CodeMirror `role="textbox"` — in shipping UI.

A page story is not optional chrome, so a new route cannot skip one. This gate
reads pageRegistry.ts for the components it routes to and requires, for each,
a `<Export>.stories.tsx` under ui/src/pages declaring that component and the
three states. No baseline and no allow-list: the set is complete today and the
gate is what keeps it complete.

Keyed on the exported component, not the file: LibraryFilesPage serves two
routes (walks and pcaps) through two exports, and a per-file check would count
one story for both.

Run locally: python3 scripts/check-page-stories.py
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

REGISTRY = Path("ui/src/pageRegistry.ts")
PAGES = Path("ui/src/pages")
REQUIRED_STORIES = ("Empty", "Loaded", "Error")

# The registry's lazy imports are the route table's own statement of which
# component serves a path — reading them means the gate cannot drift from it.
LAZY_IMPORT = re.compile(
    r"lazy\(\(\)\s*=>\s*import\('\./pages/(?P<file>[\w/]+)'\)"
    r"\.then\(\(m\)\s*=>\s*\(\{\s*default:\s*m\.(?P<export>\w+)\s*\}\)\)"
)


def routed_components(registry: str) -> dict[str, str]:
    """Map each routed component export to the page file it comes from."""
    return {m.group("export"): m.group("file") for m in LAZY_IMPORT.finditer(registry)}


def violations(components: dict[str, str]) -> list[str]:
    problems = []
    for export, file in sorted(components.items()):
        story = PAGES / f"{export}.stories.tsx"
        if not story.exists():
            problems.append(f"{export} (from {file}.tsx) has no {story}")
            continue
        text = story.read_text(encoding="utf-8")
        if f"pageMeta('{export}', {export})" not in text:
            problems.append(
                f"{story} does not build its meta with "
                f"pageMeta('{export}', {export}) — the shared harness supplies "
                "the router, providers, <main> landmark and fetch stub"
            )
        missing = [
            state
            for state in REQUIRED_STORIES
            if not re.search(rf"^export const {state}: Story", text, re.MULTILINE)
        ]
        if missing:
            problems.append(f"{story} is missing story: {', '.join(missing)}")
    return problems


def main() -> int:
    if not REGISTRY.exists():
        print(f"{REGISTRY} not found — run from the repository root", file=sys.stderr)
        return 2
    components = routed_components(REGISTRY.read_text(encoding="utf-8"))
    if not components:
        print(f"no lazy page imports found in {REGISTRY}", file=sys.stderr)
        return 2
    problems = violations(components)
    if problems:
        print("Routes without a complete page story:", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        print(
            "\nAdd ui/src/pages/<Export>.stories.tsx using pageMeta() from "
            "ui/src/test/storybook/pageStory.tsx, with Empty, Loaded and Error.",
            file=sys.stderr,
        )
        return 1
    print(f"✓ all {len(components)} routed pages have Empty/Loaded/Error stories")
    return 0


if __name__ == "__main__":
    sys.exit(main())
