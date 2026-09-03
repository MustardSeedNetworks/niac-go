#!/usr/bin/env python3
"""Self-test for check-authoring-parity.py against a throwaway tree."""

from __future__ import annotations

import importlib.util
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("parity", HERE / "check-authoring-parity.py")
parity = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(parity)

SCHEMA = {
    "$ref": "#/$defs/Config",
    "$defs": {
        "Config": {"properties": {
            "devices": {"type": "array", "items": {"$ref": "#/$defs/Device"}},
            "include_path": {"type": "string"},
            "segments": {"type": "array", "items": {"$ref": "#/$defs/Segment"}},
        }},
        "Device": {"properties": {
            "name": {"type": "string"},
            "ips": {"type": "array", "items": {"type": "string"}},
            "snmp_agent": {"$ref": "#/$defs/SnmpAgent"},
            "forward": {"type": "array", "items": {"$ref": "#/$defs/Rec"}},
            "reverse": {"type": "array", "items": {"$ref": "#/$defs/Rec"}},
        }},
        "Rec": {"properties": {"ip": {"type": "string"}}},
        "Segment": {"properties": {"devices": {"type": "array", "items": {"$ref": "#/$defs/Device"}}}},
        "SnmpAgent": {"properties": {"community": {"type": "string"}, "walk_file": {"type": "string"}}},
    },
}


class Tree:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        (self.root / "docs" / "schemas").mkdir(parents=True)
        (self.root / "ui" / "src" / "components" / "device-editor").mkdir(parents=True)
        (self.root / "scripts").mkdir()
        (self.root / "docs" / "schemas" / "niac.schema.json").write_text(json.dumps(SCHEMA))
        self.editor = self.root / "ui" / "src" / "components" / "device-editor"
        (self.editor / "SnmpSection.tsx").write_text("value={device.snmpAgent.community}")
        (self.editor / "BasicSettingsSection.tsx").write_text("value={device.hostname}")
        self.registry = self.editor / "schema-bindings.json"
        self.registry.write_text(json.dumps({
            "devices[].name": {"component": "components/device-editor/BasicSettingsSection.tsx", "field": "hostname"},
            "devices[].snmp_agent.community": "components/device-editor/SnmpSection.tsx",
        }))
        (self.root / "scripts" / "authoring-parity-allowlist.txt").write_text("include_path  # resolved by the daemon\n")

    def run(self, **kw) -> tuple[int, str]:
        out = io.StringIO()
        return parity.run(self.root, out=out, **kw), out.getvalue()


class ParityGateTest(unittest.TestCase):
    def test_leaf_walk(self) -> None:
        leaves = parity.schema_leaves(SCHEMA)
        self.assertEqual(leaves, ["devices[].forward[].ip", "devices[].ips[]", "devices[].name",
                                  "devices[].reverse[].ip", "devices[].snmp_agent.community",
                                  "devices[].snmp_agent.walk_file", "include_path"])

    def test_unbound_without_baseline_fails_then_update_passes(self) -> None:
        t = Tree()
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("devices[].snmp_agent.walk_file", out)
        self.assertEqual(t.run(update=True)[0], 0)
        code, out = t.run()
        self.assertEqual(code, 0, out)
        self.assertIn("7 schema fields, 2 bound, 1 allow-listed, 4 unbound", out)

    def test_binding_without_evidence_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.editor / "SnmpSection.tsx").write_text("nothing here")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("not evidenced", out)

    def test_binding_to_missing_file_or_unknown_path_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        t.registry.write_text(json.dumps({"devices[].name": "components/device-editor/Gone.tsx", "devices[].nope": "x"}))
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("component file missing", out)
        self.assertIn("no such field", out)

    def test_stale_baseline_entry_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.editor / "SnmpSection.tsx").write_text("community walkFile")
        reg = json.loads(t.registry.read_text())
        reg["devices[].snmp_agent.walk_file"] = "components/device-editor/SnmpSection.tsx"
        t.registry.write_text(json.dumps(reg))
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("remove them", out)


if __name__ == "__main__":
    sys.exit(unittest.main(verbosity=1))
