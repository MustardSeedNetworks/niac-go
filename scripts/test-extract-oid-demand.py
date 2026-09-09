#!/usr/bin/env python3
"""Self-test for extract-oid-demand.py against hand-built frames.

The demand matrix rests entirely on the BER decoder in that script, and a
length-encoding bug there would silently drop varbinds while still reporting
zero undecodable datagrams -- the capture cannot tell you what it did not
decode. These cases are built by hand so each decoding rule fails on its own.
"""

from __future__ import annotations

import importlib.util
import struct
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("demand", HERE / "extract-oid-demand.py")
demand = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(demand)


def tlv(tag: int, payload: bytes) -> bytes:
    if len(payload) < 0x80:
        return bytes([tag, len(payload)]) + payload
    length = len(payload).to_bytes((len(payload).bit_length() + 7) // 8, "big")
    return bytes([tag, 0x80 | len(length)]) + length + payload


def encode_oid(dotted: str) -> bytes:
    arcs = [int(arc) for arc in dotted.lstrip(".").split(".")]
    body = bytes([arcs[0] * 40 + arcs[1]])
    for arc in arcs[2:]:
        chunk = [arc & 0x7F]
        arc >>= 7
        while arc:
            chunk.append((arc & 0x7F) | 0x80)
            arc >>= 7
        body += bytes(reversed(chunk))
    return body


def snmp_request(pdu_tag: int, oids: list[str]) -> bytes:
    bindings = b"".join(tlv(0x30, tlv(0x06, encode_oid(oid)) + tlv(0x05, b"")) for oid in oids)
    pdu = tlv(0x02, b"\x01") + tlv(0x02, b"\x00") + tlv(0x02, b"\x00") + tlv(0x30, bindings)
    return tlv(0x30, tlv(0x02, b"\x01") + tlv(0x04, b"private-community") + tlv(pdu_tag, pdu))


def frame(payload: bytes, destination: str, vlan: int | None, port: int = 161) -> bytes:
    udp = struct.pack("!HHHH", 40000, port, 8 + len(payload), 0) + payload
    address = bytes(int(octet) for octet in destination.split("."))
    ip = (
        bytes([0x45, 0x00])
        + struct.pack("!H", 20 + len(udp))
        + b"\x00\x00\x00\x00\x40\x11\x00\x00"
        + b"\x0a\x00\x00\x01"
        + address
        + udp
    )
    ethernet = b"\x00" * 12
    if vlan is None:
        return ethernet + b"\x08\x00" + ip
    return ethernet + b"\x81\x00" + struct.pack("!H", vlan) + b"\x08\x00" + ip


def write_pcap(path: Path, frames: list[bytes]) -> None:
    with path.open("wb") as handle:
        handle.write(struct.pack("<IHHiIII", 0xA1B2C3D4, 2, 4, 0, 0, 262144, 1))
        for index, packet in enumerate(frames):
            handle.write(struct.pack("<IIII", 1788657251 + index, 0, len(packet), len(packet)))
            handle.write(packet)


class DecodeRequestTest(unittest.TestCase):
    def test_get_yields_its_instance(self):
        decoded = demand.decode_request(snmp_request(0xA0, [".1.3.6.1.2.1.1.3.0"]))
        self.assertEqual(decoded, [("get", ".1.3.6.1.2.1.1.3.0")])

    def test_getbulk_yields_every_varbind(self):
        decoded = demand.decode_request(
            snmp_request(0xA5, [".1.3.6.1.2.1.2.2.1.2", ".1.0.8802.1.1.2.1.4.1.1.5"])
        )
        self.assertEqual(
            decoded,
            [("getbulk", ".1.3.6.1.2.1.2.2.1.2"), ("getbulk", ".1.0.8802.1.1.2.1.4.1.1.5")],
        )

    def test_multibyte_arcs_round_trip(self):
        # 8802 and 10036 both need continuation bytes; getting the shift wrong
        # corrupts LLDP and the 802.11 MIB specifically.
        oid = ".1.2.840.10036.1.1.1.1"
        self.assertEqual(demand.decode_request(snmp_request(0xA1, [oid])), [("getnext", oid)])

    def test_long_form_length_is_read(self):
        oids = [f".1.3.6.1.2.1.2.2.1.2.{index}" for index in range(30)]
        decoded = demand.decode_request(snmp_request(0xA1, oids))
        self.assertEqual([oid for _, oid in decoded], oids)

    def test_response_pdu_is_not_demand(self):
        self.assertEqual(demand.decode_request(snmp_request(0xA2, [".1.3.6.1.2.1.1.3.0"])), [])

    def test_truncated_message_is_refused(self):
        with self.assertRaises(demand.BERError):
            demand.decode_request(snmp_request(0xA0, [".1.3.6.1.2.1.1.3.0"])[:-4])


class ExtractionTest(unittest.TestCase):
    def test_vlan_and_role_select_what_is_counted(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            capture, roles, out = root / "c.pcap", root / "roles.tsv", root / "out.tsv"
            roles.write_text("# pack\taddress\trole\nhospital\t10.51.200.5\tswitch\n")
            write_pcap(
                capture,
                [
                    frame(snmp_request(0xA0, [".1.3.6.1.2.1.1.5.0"]), "10.51.200.5", vlan=200),
                    frame(snmp_request(0xA0, [".1.3.6.1.2.1.1.5.0"]), "10.51.200.5", vlan=200),
                    # A pack VLAN, but an address the packs do not define.
                    frame(snmp_request(0xA0, [".1.3.6.1.2.1.1.5.0"]), "10.44.40.11", vlan=200),
                    # Lab infrastructure on its own VLAN.
                    frame(snmp_request(0xA0, [".1.3.6.1.2.1.1.5.0"]), "10.51.200.5", vlan=40),
                    # Untagged: no VLAN tag means no pack, same bucket as a
                    # non-pack tag. And a trap rather than a request to udp/161.
                    frame(snmp_request(0xA0, [".1.3.6.1.2.1.1.5.0"]), "10.51.200.5", vlan=None),
                    frame(snmp_request(0xA0, [".1.3.6.1.2.1.1.5.0"]), "10.51.200.5",
                          vlan=200, port=162),
                ],
            )
            demand.main(["--pcap", str(capture), "--roles", str(roles), "--out", str(out)])
            written = out.read_text()

        rows = [line for line in written.splitlines() if not line.startswith(("#", "pack\t"))]
        self.assertEqual(rows, ["hospital\tswitch\tget\t.1.3.6.1.2.1.1.5.0\t2"])
        self.assertIn("on a non-pack VLAN (lab infrastructure): 2", written)
        self.assertIn("unmapped address: 1", written)
        self.assertNotIn("private-community", written)
        self.assertNotIn("10.44.40.11", written)


if __name__ == "__main__":
    sys.exit(0 if unittest.main(exit=False).result.wasSuccessful() else 1)
