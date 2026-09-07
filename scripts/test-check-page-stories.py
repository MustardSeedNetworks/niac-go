#!/usr/bin/env python3
"""Self-test for check-page-stories.py.

A gate nobody has watched go red is indistinguishable from one that stopped
running, so CI runs this before the gate itself — the same shape as the other
ratchets. The two halves are tested separately: parsing the route table out of
pageRegistry, and deciding whether a page's story file satisfies the contract.
"""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "gate", Path(__file__).with_name("check-page-stories.py")
)
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)

REGISTRY_SAMPLE = """
const DashboardPage = lazy(() =>
  import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })),
);
const LibraryWalksPage = lazy(() =>
  import('./pages/LibraryFilesPage').then((m) => ({ default: m.LibraryWalksPage })),
);
const LibraryPcapsPage = lazy(() =>
  import('./pages/LibraryFilesPage').then((m) => ({ default: m.LibraryPcapsPage })),
);
"""

GOOD_STORY = """
const meta: Meta<typeof DashboardPage> = {
  ...pageMeta('DashboardPage', DashboardPage),
};
export const Empty: Story = { parameters: { api: EMPTY_ROUTES } };
export const Loaded: Story = { parameters: { api: LOADED_ROUTES } };
export const Error: Story = { parameters: { api: withFailure('/x') } };
"""


def check_registry_parsing() -> list[str]:
    """Two routes served by two exports of one file must count as two."""
    failures = []
    components = gate.routed_components(REGISTRY_SAMPLE)
    if components != {
        "DashboardPage": "DashboardPage",
        "LibraryWalksPage": "LibraryFilesPage",
        "LibraryPcapsPage": "LibraryFilesPage",
    }:
        failures.append(f"registry parsing returned {components}")
    return failures


def check_story_contract() -> list[str]:
    """Each way a story file can fall short must be reported."""
    cases = [
        ("a complete story satisfies the gate", GOOD_STORY, 0),
        ("a missing story file is caught", None, 1),
        (
            "a story missing the Error state is caught",
            GOOD_STORY.replace("export const Error: Story", "const unusedError"),
            1,
        ),
        (
            "a story missing Empty and Loaded is caught",
            GOOD_STORY.replace("export const Empty: Story", "const a").replace(
                "export const Loaded: Story", "const b"
            ),
            1,
        ),
        (
            "a story that hand-rolls its meta instead of using pageMeta is caught",
            GOOD_STORY.replace(
                "...pageMeta('DashboardPage', DashboardPage),",
                "title: 'Pages/DashboardPage', component: DashboardPage,",
            ),
            1,
        ),
        (
            "a story naming a different component is caught",
            GOOD_STORY.replace("pageMeta('DashboardPage', DashboardPage)", "pageMeta('X', X)"),
            1,
        ),
    ]
    failures = []
    for name, story, want in cases:
        with tempfile.TemporaryDirectory() as tmp:
            pages = Path(tmp)
            if story is not None:
                (pages / "DashboardPage.stories.tsx").write_text(story, encoding="utf-8")
            original = gate.PAGES
            gate.PAGES = pages
            try:
                got = len(gate.violations({"DashboardPage": "DashboardPage"}))
            finally:
                gate.PAGES = original
        if got != want:
            failures.append(f"{name}: {got} problems, want {want}")
    return failures


def main() -> int:
    failures = check_registry_parsing() + check_story_contract()
    for failure in failures:
        print(f"FAIL {failure}", file=sys.stderr)
    if failures:
        print(f"{len(failures)} self-test failure(s)", file=sys.stderr)
        return 1
    print("✓ check-page-stories self-test passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
