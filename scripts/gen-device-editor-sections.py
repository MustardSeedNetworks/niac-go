#!/usr/bin/env python3
"""gen-device-editor-sections.py — device-editor form manifest from the schema.

P1b-2: the device editor must be able to author everything the daemon can run
(owner decision 2026-09-02). Hand-building a form per protocol block is how it
fell 167 fields behind, so the forms are generated instead — from the same
docs/schemas/niac.schema.json that `make schema` produces from converter.Config.

This emits a typed manifest, not markup: one SectionDescriptor per device
property, each carrying the field descriptors a renderer needs (kind, enum,
bounds, description). SchemaSection.tsx walks it. Two consequences worth
naming:

  - The manifest names every field as a literal, so the authoring-parity gate
    can evidence-check a binding that points here. A renderer that walks
    `schema.properties` at runtime would contain no field names at all and
    every binding would read as unevidenced.
  - Schema `description` strings land as editor tooltips for free, which is
    the single source P1b-5 asks for.

Run: scripts/gen-device-editor-sections.py [--check]
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

SCHEMA = "docs/schemas/niac.schema.json"
OUTPUT = "ui/src/components/device-editor/generated/sections.generated.ts"
TYPES_OUTPUT = "ui/src/components/device-editor/generated/authored-device.generated.ts"

BINDINGS_OUTPUT = "ui/src/components/device-editor/schema-bindings.json"

# Device identity, kept on hand-built controls: each does more than edit a
# value. `name` is the route key, `type` drives the per-type section order, and
# `mac` / `vendor` / `mac_suffix` are one control because they are one choice --
# the daemon rejects a device carrying both (ErrDeviceMACSourceConflict), so the
# form offers the choice rather than validating it after the fact. `ips` is the
# address list the topology page also writes.
#
# Nothing else is excluded. A section with a hand-built control attached --
# snmp_agent and its synthesize-walk button -- still generates its fields, so
# the manifest stays the one owner of every value the daemon reads.
BASICS = "components/device-editor/BasicSettingsSection.tsx"

# Keyed by the Device property the manifest must skip; the value is the parity
# gate's path for it (`ips` is a list, so its leaf carries the `[]`) and the
# component that binds it.
HAND_BOUND = {
    "name": ("devices[].name", BASICS),
    "type": ("devices[].type", BASICS),
    "mac": ("devices[].mac", BASICS),
    "vendor": ("devices[].vendor", BASICS),
    "mac_suffix": ("devices[].mac_suffix", BASICS),
    "ips": ("devices[].ips[]", "components/device-editor/AdditionalIPsSection.tsx"),
}

GENERATED_COMPONENT = "components/device-editor/generated/sections.generated.ts"

# Config-level paths, which are not device properties and so are not part of the
# device manifest at all -- they are authored by the wizard. Kept here rather
# than hand-written into schema-bindings.json so the registry has exactly one
# author, and the parity gate still evidences each one against the component
# that names the field.
NETWORKS_STEP = "components/wizard/NetworksStep.tsx"
FLEET_DEFAULTS = "components/wizard/FleetDefaultsEditor.tsx"
BEHAVIOR_COMPOSER = "components/wizard/DraftBehaviorComposer.tsx"

CONFIG_BOUND = {
    "networks[].name": NETWORKS_STEP,
    "networks[].subnet": NETWORKS_STEP,
    "networks[].virtual_vlan": NETWORKS_STEP,
    "attachments[].name": NETWORKS_STEP,
    "attachments[].connect": NETWORKS_STEP,
    "capture_playbacks[].file_name": FLEET_DEFAULTS,
    "capture_playbacks[].loop_time": FLEET_DEFAULTS,
    "capture_playbacks[].scale_time": FLEET_DEFAULTS,
    **{
        f"discovery_protocols.{protocol}.{field}": FLEET_DEFAULTS
        for protocol in ("lldp", "cdp", "edp", "fdp")
        for field in ("enabled", "interval")
    },
    **{
        f"behavior_timelines[].{field}": BEHAVIOR_COMPOSER
        for field in ("name", "start_offset_ms", "repeat_count")
    },
    **{
        f"behavior_timelines[].phases[].{field}": BEHAVIOR_COMPOSER
        for field in ("name", "start_offset_ms", "duration_ms", "reset")
    },
    **{
        f"behavior_timelines[].phases[].traffic[].{field}": BEHAVIOR_COMPOSER
        for field in ("device", "interface", "utilization")
    },
    **{
        f"behavior_timelines[].phases[].faults[].{field}": BEHAVIOR_COMPOSER
        for field in ("device", "interface", "type", "value")
    },
}

HAND_BOUND_PATHS = dict(HAND_BOUND.values())

TITLES = {
    "cdp": "CDP",
    "dhcp": "DHCP",
    "dhcpv6": "DHCPv6",
    "dns": "DNS",
    "edp": "EDP",
    "fdp": "FDP",
    "ftp": "FTP",
    "http": "HTTP",
    "icmp": "ICMP",
    "icmpv6": "ICMPv6",
    "iperf3": "iPerf3",
    "lldp": "LLDP",
    "mac_suffix": "MAC suffix",
    "map_to_ip": "Map to IP",
    "netbios": "NetBIOS",
    "os_fingerprint": "OS fingerprint",
    "port_channels": "Port channels",
    "snmpv3": "SNMPv3",
    "ssh": "SSH",
    "stp": "STP",
    "trunk_ports": "Trunk ports",
    "ttl": "TTL",
}


def title_of(name: str) -> str:
    if name in TITLES:
        return TITLES[name]
    return name.replace("_", " ").capitalize()


def resolve(node: dict, defs: dict) -> dict:
    while "$ref" in node:
        node = defs[node["$ref"].rsplit("/", 1)[-1]]
    return node


def describe(name: str, node: dict, defs: dict, stack: tuple[str, ...]) -> dict | None:
    """One field descriptor, or None for a recursive or unrenderable node."""
    ref = node.get("$ref", "")
    if ref in stack:
        return None
    stack = stack + ((ref,) if ref else ())
    resolved = resolve(node, defs)

    field: dict = {"name": name, "title": title_of(name)}
    if resolved.get("description"):
        field["description"] = resolved["description"]
    for bound in ("minimum", "maximum", "pattern"):
        if bound in resolved:
            field[bound] = resolved[bound]
    if resolved.get("enum"):
        field["kind"] = "enum"
        field["options"] = resolved["enum"]
        return field

    kind = resolved.get("type")
    if kind == "array":
        items = resolved.get("items", {})
        item = resolve(items, defs)
        if item.get("properties"):
            field["kind"] = "objectList"
            field["fields"] = child_fields(item, defs, stack)
            return field
        field["kind"] = "scalarList"
        field["itemKind"] = item.get("type", "string")
        return field
    if kind == "object" or resolved.get("properties"):
        if resolved.get("properties"):
            field["kind"] = "object"
            field["fields"] = child_fields(resolved, defs, stack)
            return field
        # additionalProperties-only: a free-form string map (Properties).
        field["kind"] = "map"
        return field
    if kind in ("string", "boolean", "integer", "number"):
        field["kind"] = kind
        return field

    return None


def child_fields(node: dict, defs: dict, stack: tuple[str, ...]) -> list[dict]:
    out = []
    for name, child in node.get("properties", {}).items():
        described = describe(name, child, defs, stack)
        if described is not None:
            out.append(described)
    return out


def build(schema: dict) -> list[dict]:
    defs = schema.get("$defs", {})
    device = defs["Device"]
    scalars: list[dict] = []
    sections: list[dict] = []

    for name, node in device["properties"].items():
        if name in HAND_BOUND:
            continue
        field = describe(name, node, defs, ())
        if field is None:
            continue
        if field["kind"] in ("object", "objectList", "map"):
            sections.append({"key": name, "title": title_of(name), "fields": field.get("fields", []),
                             "kind": field["kind"]})
        else:
            scalars.append(field)

    if scalars:
        sections.insert(0, {"key": "device", "title": "Device", "fields": scalars, "kind": "object"})
    return sections


def device_types(schema: dict) -> list[str]:
    """The `type` vocabulary. Hand-bound, so it is absent from the manifest,
    but the editor's picker still has to offer the daemon's own words -- the
    UI's DeviceType union is a third vocabulary that matches neither."""
    defs = schema.get("$defs", {})
    return resolve(defs["Device"]["properties"]["type"], defs).get("enum", [])


def binding_paths(sections: list[dict]) -> list[str]:
    """Every settable leaf in the manifest, as the parity gate spells it."""
    paths: list[str] = []

    def walk(field: dict, path: str) -> None:
        kind = field["kind"]
        if kind == "object":
            for child in field.get("fields", []):
                walk(child, f"{path}.{child['name']}")
            return
        if kind == "objectList":
            for child in field.get("fields", []):
                walk(child, f"{path}[].{child['name']}")
            return
        paths.append(f"{path}[]" if kind == "scalarList" else path)

    for section in sections:
        if section["key"] == "device":
            for field in section["fields"]:
                walk(field, f"devices[].{field['name']}")
            continue
        walk({**section, "name": section["key"]}, f"devices[].{section['key']}")

    return paths


def render_bindings(sections: list[dict]) -> str:
    """The parity gate's registry.

    Generated rather than hand-listed: the manifest is what the editor renders,
    so it is also the honest evidence that a field is reachable. Hand-typing 217
    entries would let a field the generator dropped stay quietly bound.
    """
    bindings: dict[str, object] = {**HAND_BOUND_PATHS, **CONFIG_BOUND}
    for path in binding_paths(sections):
        bindings[path] = GENERATED_COMPONENT
    return json.dumps(dict(sorted(bindings.items())), indent=2) + "\n"


HEADER = """/**
 * GENERATED by scripts/gen-device-editor-sections.py from
 * docs/schemas/niac.schema.json. Do not edit — run `make ui-sections`.
 *
 * The device editor's form definition. Every field the daemon's authored
 * Device carries appears here exactly once, named, so SchemaSection can render
 * it and the authoring-parity gate can evidence a binding against it.
 */

export type FieldKind =
  | 'string'
  | 'boolean'
  | 'integer'
  | 'number'
  | 'enum'
  | 'scalarList'
  | 'objectList'
  | 'object'
  | 'map';

export interface FieldDescriptor {
  readonly name: string;
  readonly title: string;
  readonly kind: FieldKind;
  readonly description?: string;
  readonly options?: readonly string[];
  readonly itemKind?: string;
  readonly minimum?: number;
  readonly maximum?: number;
  readonly pattern?: string;
  readonly fields?: readonly FieldDescriptor[];
}

export interface SectionDescriptor {
  readonly key: string;
  readonly title: string;
  readonly kind: FieldKind;
  readonly fields: readonly FieldDescriptor[];
}

export const DEVICE_SECTIONS: readonly SectionDescriptor[] = """


def render(sections: list[dict], types: list[str]) -> str:
    body = json.dumps(sections, indent=2)
    vocabulary = json.dumps(types, indent=2)

    return (
        f"{HEADER}{body} as const;\n\n"
        "/** The daemon's own `type` vocabulary, for the hand-built type picker. */\n"
        f"export const DEVICE_TYPES: readonly string[] = {vocabulary} as const;\n"
    )


TS_SCALARS = {
    "string": "string",
    "boolean": "boolean",
    "integer": "number",
    "number": "number",
}


def ts_type(field: dict, indent: str) -> str:
    """The TypeScript type for one field descriptor."""
    kind = field["kind"]
    if kind == "enum":
        return " | ".join(f"'{option}'" for option in field["options"])
    if kind == "map":
        return "Record<string, string>"
    if kind == "scalarList":
        return f"readonly {TS_SCALARS.get(field.get('itemKind', 'string'), 'string')}[]"
    if kind in ("object", "objectList"):
        body = ts_object(field.get("fields", []), indent + "  ")
        return f"readonly {body}[]" if kind == "objectList" else body
    return TS_SCALARS[kind]


def ts_object(fields: list[dict], indent: str) -> str:
    if not fields:
        return "Record<string, never>"
    lines = ["{"]
    for field in fields:
        lines.append(f"{indent}  readonly {field['name']}?: {ts_type(field, indent + '  ')};")
    lines.append(indent + "}")
    return "\n".join(lines)


TYPES_HEADER = """/**
 * GENERATED by scripts/gen-device-editor-sections.py from
 * docs/schemas/niac.schema.json. Do not edit — run `make ui-sections`.
 *
 * The device as an author writes it: the daemon's own snake_case YAML shape,
 * not the camelCase API projection. The editor loads and saves this document
 * verbatim through `rawYaml`, which is what makes its round trip an identity
 * rather than a mapping that can drop fields.
 */

/** Any value an authored YAML document can hold. */
export type AuthoredValue =
  | string
  | number
  | boolean
  | undefined
  | readonly AuthoredValue[]
  | { readonly [key: string]: AuthoredValue };

export interface AuthoredDevice """


def render_types(schema: dict) -> str:
    defs = schema.get("$defs", {})
    device = defs["Device"]
    fields = []
    for name, node in device["properties"].items():
        described = describe(name, node, defs, ())
        if described is not None:
            fields.append(described)
    return TYPES_HEADER + ts_object(fields, "") + "\n"


def artifacts(root: Path) -> dict[str, str]:
    """What the committed generated files should contain right now."""
    schema = json.loads((root / SCHEMA).read_text(encoding="utf-8"))

    sections = build(schema)

    return {
        OUTPUT: render(sections, device_types(schema)),
        TYPES_OUTPUT: render_types(schema),
        BINDINGS_OUTPUT: render_bindings(sections),
    }


def run_check(root: Path) -> int:
    stale = [
        name
        for name, rendered in artifacts(root).items()
        if ((root / name).read_text(encoding="utf-8") if (root / name).exists() else "") != rendered
    ]
    for name in stale:
        print(f"::error::{name} is stale — run `make ui-sections` and commit the result")
    if stale:
        return 1

    print("Device-editor manifest and types are up to date.")
    return 0


def run_write(root: Path) -> int:
    for name, rendered in artifacts(root).items():
        out = root / name
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(rendered, encoding="utf-8")
        print(f"Wrote {name}")

    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--check", action="store_true", help="fail if the committed manifest is stale")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    args = parser.parse_args()

    return run_check(args.root) if args.check else run_write(args.root)


if __name__ == "__main__":
    sys.exit(main())
