#!/usr/bin/env python3
"""Tests for gen-device-editor-sections.py.

The manifest is what the device editor's forms are built from and what the
authoring-parity gate evidences bindings against, so the invariant that matters
is coverage: every field of the daemon's authored Device has to be reachable in
the manifest or on a hand-built control, with no field owned by both.

This used to compare against the parity baseline's device rows. That was only
meaningful while the baseline had any: it is now empty of them, so the schema
itself is the comparison.

Run: scripts/test-gen-device-editor-sections.py
"""

from __future__ import annotations

import importlib.util
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

spec = importlib.util.spec_from_file_location("gen", ROOT / "scripts" / "gen-device-editor-sections.py")
gen = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gen)


def manifest_leaves(sections: list[dict]) -> set[str]:
    """Every settable path in the manifest, in the baseline's dotted notation."""
    leaves: set[str] = set()

    def walk(prefix: str, fields: list[dict]) -> None:
        for field in fields:
            path = f"{prefix}.{field['name']}" if prefix else field["name"]
            if field["kind"] == "objectList":
                walk(f"{path}[]", field.get("fields", []))
            elif field["kind"] == "object":
                walk(path, field.get("fields", []))
            elif field["kind"] == "scalarList":
                leaves.add(f"{path}[]")
            else:
                leaves.add(path)

    for section in sections:
        if section["kind"] == "map":
            leaves.add(section["key"])
            continue
        prefix = "" if section["key"] == "device" else section["key"]
        if section["kind"] == "objectList":
            prefix = f"{prefix}[]"
        walk(prefix, section["fields"])

    return leaves


def schema_device_fields() -> list[str]:
    """Every device leaf the parity gate walks, minus the `devices[].` prefix."""
    spec = importlib.util.spec_from_file_location("gate", ROOT / "scripts" / "check-authoring-parity.py")
    gate = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(gate)
    schema = json.loads((ROOT / gen.SCHEMA).read_text(encoding="utf-8"))

    return [
        leaf[len("devices[]."):]
        for leaf in gate.schema_leaves(schema)
        if leaf.startswith("devices[].")
    ]


def test_manifest_covers_every_device_field(sections: list[dict]) -> list[str]:
    leaves = manifest_leaves(sections)
    hand_bound = {path[len("devices[]."):] for path in gen.HAND_BOUND_PATHS}
    missing = [
        field
        for field in schema_device_fields()
        if field not in leaves and field not in hand_bound
    ]

    return [f"no form can set `{field}`: neither generated nor hand-bound" for field in missing]


def test_identity_fields_stay_hand_bound(sections: list[dict]) -> list[str]:
    top = {field["name"] for section in sections if section["key"] == "device" for field in section["fields"]}
    clashes = sorted(set(gen.HAND_BOUND) & (top | {section["key"] for section in sections}))
    return [f"`{field}` is hand-bound but also generated — two owners for one value" for field in clashes]


def test_every_section_has_fields(sections: list[dict]) -> list[str]:
    return [
        f"section `{section['key']}` renders nothing"
        for section in sections
        if not section["fields"] and section["kind"] != "map"
    ]


def test_committed_output_is_current() -> list[str]:
    return [] if gen.run_check(ROOT) == 0 else ["committed manifest or types are stale"]


def main() -> int:
    schema = json.loads((ROOT / gen.SCHEMA).read_text(encoding="utf-8"))
    sections = gen.build(schema)

    failures = (
        test_manifest_covers_every_device_field(sections)
        + test_identity_fields_stay_hand_bound(sections)
        + test_every_section_has_fields(sections)
        + test_committed_output_is_current()
    )
    for failure in failures:
        print(f"::error::{failure}")
    if failures:
        return 1

    print(
        f"gen-device-editor-sections: {len(sections)} sections, "
        f"{len(manifest_leaves(sections))} settable paths, "
        f"all {len(schema_device_fields())} device fields covered."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
