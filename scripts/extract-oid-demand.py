#!/usr/bin/env python3
"""Extract the OID demand of a discovery instrument from a packet capture.

Plan row F3(b). NIAC's fidelity harness (F1a) measures what a walk *serves*.
This measures the other side: what a real NMS actually *asks for*, so the two
can be joined into a coverage report. The input is one tcpdump of a discovery
run against the shipped packs; the output is committed metadata only -- pack,
role, PDU type, OID, request count. Addresses, community strings and payload
values never leave this script.

The capture is large (a six-pack EtherScope discovery is ~250 MB) and lives on
the lab host, which has neither tshark nor scapy, so the pcap reader and the
SNMP BER decoder here are deliberately stdlib-only: the script is copied to the
capture host and run there.

Usage:
  python3 extract-oid-demand.py --pcap <file> --roles <role map> --out <tsv>

The role map comes from the scenario generator, not by hand:
  NIAC_ROLE_MAP_OUT=roles.tsv go test ./internal/scenario -run TestWritePackDeviceRoles
"""

import argparse
import collections
import datetime
import struct
import sys

# Physical VLAN assignment, from docs/design/2026-08-linklive-acceptance-runbook.md.
# Deployment identity, deliberately not stored inside a portable pack, which is
# why it is repeated here rather than read out of one.
VLAN_PACKS = {
    200: "hospital",
    201: "warehouse",
    202: "manufacturing",
    203: "campus",
    204: "retail",
    205: "service-provider",
    299: "enterprise-scale",
}

SNMP_PORT = 161
PDU_NAMES = {0xA0: "get", 0xA1: "getnext", 0xA5: "getbulk", 0xA3: "set"}


class BERError(ValueError):
    """A datagram that does not decode as the SNMP message we expect."""


def read_length(buf, index):
    first = buf[index]
    index += 1
    if first < 0x80:
        return first, index
    count = first & 0x7F
    if count == 0 or count > 4 or index + count > len(buf):
        raise BERError("unsupported length encoding")
    return int.from_bytes(buf[index : index + count], "big"), index + count


def read_tlv(buf, index):
    if index >= len(buf):
        raise BERError("truncated tag")
    tag = buf[index]
    length, index = read_length(buf, index + 1)
    if index + length > len(buf):
        raise BERError("truncated value")
    return tag, buf[index : index + length], index + length


def decode_oid(raw):
    """BER object identifier -> dotted form, the same shape the walk files use."""
    if not raw:
        raise BERError("empty object identifier")
    arcs = [str(raw[0] // 40), str(raw[0] % 40)]
    value = 0
    for byte in raw[1:]:
        value = (value << 7) | (byte & 0x7F)
        if not byte & 0x80:
            arcs.append(str(value))
            value = 0
    return "." + ".".join(arcs)


def decode_request(payload):
    """Yield (pdu name, oid) for one SNMP v1/v2c request datagram.

    Returns an empty list for a response or a PDU type this does not model; the
    caller counts those separately rather than guessing.
    """
    tag, message, _ = read_tlv(payload, 0)
    if tag != 0x30:
        raise BERError("message is not a SEQUENCE")
    index = 0
    _, _, index = read_tlv(message, index)  # version
    _, _, index = read_tlv(message, index)  # community -- read past, never kept
    pdu_tag, pdu, _ = read_tlv(message, index)
    name = PDU_NAMES.get(pdu_tag)
    if name is None:
        return []
    index = 0
    for _ in range(3):  # request-id, error-status/non-repeaters, error-index/max-repetitions
        _, _, index = read_tlv(pdu, index)
    bindings_tag, bindings, _ = read_tlv(pdu, index)
    if bindings_tag != 0x30:
        raise BERError("varbind list is not a SEQUENCE")
    requested = []
    offset = 0
    while offset < len(bindings):
        _, binding, offset = read_tlv(bindings, offset)
        oid_tag, oid, _ = read_tlv(binding, 0)
        if oid_tag == 0x06:
            requested.append((name, decode_oid(oid)))
    return requested


def snmp_datagrams(path):
    """Yield (epoch seconds, vlan, destination, payload) for UDP/161 traffic."""
    with open(path, "rb") as capture:
        header = capture.read(24)
        if len(header) < 24:
            raise SystemExit(f"{path}: not a pcap file")
        if header[:4] == b"\xd4\xc3\xb2\xa1":
            endian = "<"
        elif header[:4] == b"\xa1\xb2\xc3\xd4":
            endian = ">"
        else:
            raise SystemExit(f"{path}: unsupported capture format (pcapng is not handled)")
        while True:
            record = capture.read(16)
            if len(record) < 16:
                return
            seconds, _micros, captured, _original = struct.unpack(endian + "IIII", record)
            packet = capture.read(captured)
            if len(packet) < captured or captured < 34:
                return
            ethertype = struct.unpack("!H", packet[12:14])[0]
            offset = 14
            vlan = 0
            while ethertype in (0x8100, 0x88A8):
                if len(packet) < offset + 4:
                    break
                vlan = struct.unpack("!H", packet[offset : offset + 2])[0] & 0x0FFF
                ethertype = struct.unpack("!H", packet[offset + 2 : offset + 4])[0]
                offset += 4
            if ethertype != 0x0800 or len(packet) < offset + 20:
                continue
            if packet[offset + 9] != 17:  # UDP
                continue
            destination = ".".join(str(octet) for octet in packet[offset + 16 : offset + 20])
            udp = offset + (packet[offset] & 0x0F) * 4
            if len(packet) < udp + 8:
                continue
            if struct.unpack("!H", packet[udp + 2 : udp + 4])[0] != SNMP_PORT:
                continue
            yield seconds, vlan, destination, packet[udp + 8 :]


def load_roles(path):
    roles = {}
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            if line.startswith("#") or not line.strip():
                continue
            pack, address, role = line.rstrip("\n").split("\t")
            roles[(pack, address)] = role
    return roles


def utc(seconds):
    return datetime.datetime.fromtimestamp(seconds, datetime.UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--pcap", required=True)
    parser.add_argument("--roles", required=True, help="pack/address/role map from TestWritePackDeviceRoles")
    parser.add_argument("--consumer", default="etherscope-nxg", help="the instrument that produced the capture")
    parser.add_argument("--out", required=True)
    arguments = parser.parse_args(argv)

    roles = load_roles(arguments.roles)
    demand = collections.Counter()
    first = last = None
    datagrams = decoded = undecodable = off_pack = unmapped = 0
    unmapped_addresses = set()

    for seconds, vlan, destination, payload in snmp_datagrams(arguments.pcap):
        datagrams += 1
        first = seconds if first is None else min(first, seconds)
        last = seconds if last is None else max(last, seconds)
        pack = VLAN_PACKS.get(vlan)
        if pack is None:
            off_pack += 1
            continue
        role = roles.get((pack, destination))
        if role is None:
            unmapped += 1
            unmapped_addresses.add((pack, destination))
            continue
        try:
            requested = decode_request(payload)
        except BERError:
            undecodable += 1
            continue
        if requested:
            decoded += 1
        for name, oid in requested:
            demand[(pack, role, name, oid)] += 1

    with open(arguments.out, "w", encoding="utf-8") as out:
        out.write(f"# Consumer OID demand matrix -- plan row F3(b).\n")
        out.write(f"# Consumer: {arguments.consumer}\n")
        out.write(f"# Capture window (UTC): {utc(first)} .. {utc(last)}\n")
        out.write(f"# SNMP request datagrams to udp/{SNMP_PORT}: {datagrams}\n")
        out.write(f"# ... on a pack VLAN and a mapped pack address: {decoded}\n")
        out.write(f"# ... on a non-pack VLAN (lab infrastructure): {off_pack}\n")
        out.write(f"# ... on a pack VLAN but an unmapped address: {unmapped}"
                  f" across {len(unmapped_addresses)} addresses\n")
        out.write(f"# ... undecodable as an SNMP request: {undecodable}\n")
        out.write("# Regenerate:\n")
        out.write("#   NIAC_ROLE_MAP_OUT=roles.tsv go test ./internal/scenario -run TestWritePackDeviceRoles\n")
        out.write("#   python3 scripts/extract-oid-demand.py --pcap <capture> --roles roles.tsv --out <this file>\n")
        out.write("pack\trole\tpdu\toid\trequests\n")
        for (pack, role, name, oid), count in sorted(demand.items()):
            out.write(f"{pack}\t{role}\t{name}\t{oid}\t{count}\n")

    print(f"{arguments.out}: {len(demand)} rows from {decoded} request datagrams", file=sys.stderr)
    if unmapped_addresses:
        print(f"warning: {len(unmapped_addresses)} pack-VLAN addresses are not in the role map",
              file=sys.stderr)


if __name__ == "__main__":
    main(sys.argv[1:])
