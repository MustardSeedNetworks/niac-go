#!/usr/bin/env python3
"""check-authoring-parity.py — YAML schema ⇄ device editor parity gate.

Owner decision 2026-09-02: the YAML file, the device editor and the wizard
must each be able to author everything the daemon can run. The daemon's
authoring surface is docs/schemas/niac.schema.json (generated from
converter.Config by `make schema`). This gate walks every leaf field in that
schema and asks whether the UI can set it.

A field is BOUND when ui/src/components/device-editor/schema-bindings.json
maps its path to a component file that exists and mentions the field by its
snake_case or camelCase name — the registry is evidence-checked, not trusted.
A field is ALLOWED when scripts/authoring-parity-allowlist.txt names it with
a reason (server-computed, derived from another field). Everything else is
UNBOUND and is a RATCHET against scripts/authoring-parity-baseline.txt:
growth fails, and a baselined field that has since been bound fails too, so
the baseline is the P1b-2 work queue rather than an allow-list.

Registry entries that point at a missing file, lack evidence, or name a path
the schema no longer has fail outright.

Run locally: scripts/check-authoring-parity.py   ·   --list · --update
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

SCHEMA = "docs/schemas/niac.schema.json"
REGISTRY = "ui/src/components/device-editor/schema-bindings.json"
ALLOWLIST = "scripts/authoring-parity-allowlist.txt"
BASELINE = "scripts/authoring-parity-baseline.txt"


def _resolve(node: dict, defs: dict) -> dict:
    while "$ref" in node:
        name = node["$ref"].rsplit("/", 1)[-1]
        node = defs[name]
    return node


def schema_leaves(schema: dict) -> list[str]:
    defs = schema.get("$defs", {})
    root = _resolve(schema, defs)
    leaves: list[str] = []
    first_path: dict[str, str] = {}

    def walk(node: dict, path: str, stack: tuple[str, ...]) -> None:
        ref = node.get("$ref", "")
        if ref and ref in stack:  # cyclic definition, e.g. include_path recursion
            leaves.append(path)
            return
        if ref:
            # The same definition under the same property name at a second
            # depth (segments[].devices[] is the Device list of devices[]) is
            # one authoring surface: one binding covers both. The same
            # definition under a different property (forward_records vs
            # reverse_records) is two surfaces and both are walked.
            earlier = first_path.get(ref)
            if earlier is not None and earlier.rsplit(".", 1)[-1] == path.rsplit(".", 1)[-1] and earlier != path:
                return
            first_path.setdefault(ref, path)
        stack = stack + ((ref,) if ref else ())
        node = _resolve(node, defs)
        props = node.get("properties")
        if props:
            for key, child in props.items():
                walk(child, f"{path}.{key}" if path else key, stack)
            return
        if node.get("type") == "array" and isinstance(node.get("items"), dict):
            items = node["items"]
            resolved = _resolve(items, defs)
            if resolved.get("properties"):
                walk(items, f"{path}[]", stack)
            else:
                leaves.append(f"{path}[]")
            return
        leaves.append(path)

    walk(root, "", ())
    return sorted(leaves)


def camel(snake: str) -> str:
    head, *rest = snake.split("_")
    return head + "".join(part.capitalize() for part in rest)


def read_listfile(path: Path) -> dict[str, str]:
    entries: dict[str, str] = {}
    if path.exists():
        for raw in path.read_text(encoding="utf-8").splitlines():
            line = raw.strip()
            if line and not line.startswith("#"):
                key, _, reason = line.partition("#")
                entries[key.strip()] = reason.strip()
    return entries


def evidence(root: Path, binding: str | dict, leaf_path: str) -> str | None:
    """A binding is a component path, or {"component": path, "field": name}
    when the editor calls the field something else (schema `name` is the
    editor's `hostname`)."""
    component = binding["component"] if isinstance(binding, dict) else binding
    file = root / "ui" / "src" / component
    if not file.is_file():
        return f"component file missing: {component}"
    leaf = leaf_path.rstrip("[]").rsplit(".", 1)[-1]
    names = [binding["field"]] if isinstance(binding, dict) else [leaf, camel(leaf)]
    text = file.read_text(encoding="utf-8")
    if not any(re.search(rf"\b{re.escape(n)}\b", text, re.IGNORECASE) for n in names):
        return f"{component} never mentions {' or '.join(f'`{n}`' for n in names)}"
    return None


def run(root: Path, update: bool = False, list_only: bool = False, out=sys.stdout) -> int:
    schema = json.loads((root / SCHEMA).read_text(encoding="utf-8"))
    leaves = schema_leaves(schema)
    if list_only:
        print("\n".join(leaves), file=out)
        return 0
    registry_path = root / REGISTRY
    registry: dict[str, str | dict] = json.loads(registry_path.read_text(encoding="utf-8")) if registry_path.exists() else {}
    allowed = read_listfile(root / ALLOWLIST)
    previous = read_listfile(root / BASELINE)
    leaf_set = set(leaves)

    failed = False
    for path, component in sorted(registry.items()):
        if path not in leaf_set:
            failed = True
            print(f"::error::registry binds `{path}` but the schema has no such field", file=out)
            continue
        problem = evidence(root, component, path)
        if problem:
            failed = True
            print(f"::error::binding for `{path}` is not evidenced: {problem}", file=out)
    for path in allowed:
        if path not in leaf_set:
            failed = True
            print(f"::error::allowlist names `{path}` but the schema has no such field", file=out)

    unbound = sorted(p for p in leaves if p not in registry and p not in allowed)
    if update:
        lines = ["# Schema fields the device editor cannot set yet (ratchet; see check-authoring-parity.py).",
                 "# Bind a field in schema-bindings.json, or allow-list it with a reason, then remove it here."]
        lines += [f"{p}  # {previous[p]}" if previous.get(p) else p for p in unbound]
        (root / BASELINE).write_text("\n".join(lines) + "\n", encoding="utf-8")
        print(f"Wrote {len(unbound)} unbound field(s) to {BASELINE}", file=out)
        return 0

    new = [p for p in unbound if p not in previous]
    stale = [p for p in previous if p not in unbound]
    if new:
        failed = True
        print("::error::schema fields with no editor binding that are not in the baseline "
              "(bind them, allow-list them with a reason, or baseline them):", file=out)
        for p in new:
            print(f"  {p}", file=out)
    if stale:
        failed = True
        print(f"::error::baseline entries now bound, allowed, or gone — remove them from {BASELINE}:", file=out)
        for p in stale:
            print(f"  {p}", file=out)
    bound = len(leaf_set) - len(unbound) - sum(1 for p in allowed if p in leaf_set)
    print(f"Authoring-parity gate: {len(leaves)} schema fields, {bound} bound, "
          f"{sum(1 for p in allowed if p in leaf_set)} allow-listed, {len(unbound)} unbound "
          f"({len(previous)} baselined).", file=out)
    return 1 if failed else 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--update", action="store_true", help="rewrite the baseline from the current tree")
    parser.add_argument("--list", action="store_true", help="print every schema leaf path and exit")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    args = parser.parse_args()
    return run(args.root, update=args.update, list_only=args.list)


if __name__ == "__main__":
    sys.exit(main())
