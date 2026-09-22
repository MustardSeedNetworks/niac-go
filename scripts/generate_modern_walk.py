#!/usr/bin/env python3
"""
Generate synthetic SNMP walk files for modern network devices.

This tool creates realistic SNMP walk files based on vendor specifications
and MIB documentation for modern network equipment.

Usage:
    python3 generate_modern_walk.py --vendor cisco --model c3850-48p --output walk.snmp
"""

import argparse
import sys
from datetime import datetime

# Device templates with realistic OID responses.
#
# `sysobjectid` and `sysservices` are per model, never per vendor and never
# defaulted. The enterprise arc of each sysObjectID is the vendor's IANA
# private enterprise number (enterprise-numbers.txt, 2026-09-17): Cisco 9,
# HP 11, Extreme 1916, Juniper 2636, Dell 674, Fortinet 12356, Palo Alto
# 25461, Meraki 29671, Arista 30065, HPE (ArubaOS-CX) 47196. `scripts/
# check-starter-walks.sh` enforces exactly that arc against each walk's
# sysDescr, so a new template with the wrong one cannot ship.
#
# Below the enterprise arc these are the vendor's documented product branch,
# not a model-specific leaf, except where the leaf is sourced (the QFX5100-48S
# and Catalyst 9300-48P values are). Inventing a precise-looking leaf we
# cannot verify would repeat #2154 in a form that is harder to notice.
#
# `sysservices` is the RFC 1213 bit sum (1 physical, 2 datalink, 4 internet,
# 8 end-to-end, 64 applications). The modelled switches are L3-capable, so
# they report 6 -- the value the captured Cisco C3560/C3750 walks in this
# corpus report -- and the PA-440 firewall reports 76 (internet, end-to-end,
# applications). The old universal 78 claimed application-layer services for
# every access switch.
DEVICE_TEMPLATES = {
    "cisco": {
        "c3850-48p": {
            "model": "WS-C3850-48P",
            "sysobjectid": ".1.3.6.1.4.1.9.1",
            "sysservices": 6,
            "description": "Cisco IOS Software, C3850 Software (C3850-UNIVERSALK9-M), Version 16.12.4, RELEASE SOFTWARE (fc5)",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": [
                "TenGigabitEthernet1/1/1",
                "TenGigabitEthernet1/1/2",
                "TenGigabitEthernet1/1/3",
                "TenGigabitEthernet1/1/4",
            ],
        },
        "c3650-48p": {
            "model": "WS-C3650-48PD",
            "sysobjectid": ".1.3.6.1.4.1.9.1",
            "sysservices": 6,
            "description": "Cisco IOS Software, C3650 Software (C3650-UNIVERSALK9-M), Version 16.12.4, RELEASE SOFTWARE (fc5)",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["TenGigabitEthernet1/1/1", "TenGigabitEthernet1/1/2"],
        },
        "c9300-48p": {
            "model": "C9300-48P",
            "sysobjectid": ".1.3.6.1.4.1.9.1.2697",
            "sysservices": 6,
            "description": "Cisco IOS Software [Gibraltar], Catalyst L3 Switch Software (CAT9K_IOSXE), Version 17.6.3, RELEASE SOFTWARE (fc4)",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": [
                "TenGigabitEthernet1/1/1",
                "TenGigabitEthernet1/1/2",
                "TenGigabitEthernet1/1/3",
                "TenGigabitEthernet1/1/4",
            ],
        },
        "n9k-c9300": {
            "model": "N9K-C9300",
            "sysobjectid": ".1.3.6.1.4.1.9.12.3.1.3",
            "sysservices": 6,
            "description": "Cisco Nexus Operating System (NX-OS) Software, Version 9.3(8)",
            "ports": 32,
            "stacking": False,
            "poe": False,
            "uplinks": ["Ethernet1/1", "Ethernet1/2", "Ethernet1/3", "Ethernet1/4"],
        },
    },
    "juniper": {
        "ex4300-48p": {
            "model": "EX4300-48P",
            "sysobjectid": ".1.3.6.1.4.1.2636.1.1.1.2",
            "sysservices": 6,
            "description": "Juniper Networks, Inc. ex4300-48p Ethernet Switch, kernel JUNOS 20.4R3.8, Build date: 2021-10-15",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["ge-0/1/0", "ge-0/1/1", "ge-0/1/2", "ge-0/1/3"],
        },
        "qfx5100-48s": {
            "model": "QFX5100-48S",
            "sysobjectid": ".1.3.6.1.4.1.2636.1.1.1.4.82.5",
            "sysservices": 6,
            "description": "Juniper Networks, Inc. qfx5100-48s-6q Ethernet Switch, kernel JUNOS 20.4R3.8",
            "ports": 48,
            "stacking": False,
            "poe": False,
            "uplinks": ["et-0/0/48", "et-0/0/49", "et-0/0/50", "et-0/0/51"],
        },
    },
    "aruba": {
        "cx6300-48g": {
            "model": "JL762A",
            "sysobjectid": ".1.3.6.1.4.1.47196.4.1.1.3",
            "sysservices": 6,
            "description": "Aruba CX 6300 48G 4SFP56 Switch, ArubaOS-CX FL.10.10.1010",
            "ports": 48,
            "stacking": True,
            "poe": False,
            "uplinks": ["1/1/49", "1/1/50", "1/1/51", "1/1/52"],
        },
        "cx8360-48y8c": {
            "model": "JL706A",
            "sysobjectid": ".1.3.6.1.4.1.47196.4.1.1.3",
            "sysservices": 6,
            "description": "Aruba CX 8360-48Y8C Switch, ArubaOS-CX FL.10.10.1010",
            "ports": 48,
            "stacking": False,
            "poe": False,
            "uplinks": ["1/1/49", "1/1/50", "1/1/51", "1/1/52"],
        },
    },
    "extreme": {
        "x465-48w": {
            "model": "X465-48W",
            "sysobjectid": ".1.3.6.1.4.1.1916.2",
            "sysservices": 6,
            "description": "ExtremeXOS (X465-48W) version 32.3.1.4 by release-manager on Thu Nov  4 19:18:03 EDT 2021",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["1:49", "1:50", "1:51", "1:52"],
        },
    },
    "paloalto": {
        "pa-440": {
            "model": "PA-440",
            "sysobjectid": ".1.3.6.1.4.1.25461.2.3",
            "sysservices": 76,
            "description": "Palo Alto Networks PA-440 firewall, PAN-OS 10.2.3",
            "ports": 8,
            "stacking": False,
            "poe": False,
            "uplinks": ["ethernet1/1", "ethernet1/2"],
        },
    },
    "hpe": {
        "aruba-2930f-48g": {
            "model": "JL262A",
            "sysobjectid": ".1.3.6.1.4.1.11.2.3.7.11",
            "sysservices": 6,
            "description": "HPE Aruba 2930F 48G 4SFP+ Switch, FL.16.11.0009",
            "ports": 48,
            "stacking": True,
            "poe": False,
            "uplinks": ["1/49", "1/50", "1/51", "1/52"],
        },
        "aruba-6300m-48g": {
            "model": "JL659A",
            "sysobjectid": ".1.3.6.1.4.1.47196.4.1.1.3",
            "sysservices": 6,
            "description": "HPE Aruba CX 6300M 48G Class 4 PoE 4SFP56 Switch, ArubaOS-CX FL.10.11.1020",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["1/1/49", "1/1/50", "1/1/51", "1/1/52"],
        },
    },
    "arista": {
        "7050sx3-48yc12": {
            "model": "7050SX3-48YC12",
            "sysobjectid": ".1.3.6.1.4.1.30065.1",
            "sysservices": 6,
            "description": "Arista Networks EOS version 4.28.3M",
            "ports": 48,
            "stacking": False,
            "poe": False,
            "uplinks": ["Ethernet49/1", "Ethernet50/1", "Ethernet51/1", "Ethernet52/1"],
        },
        "7280sr3-48yc8": {
            "model": "7280SR3-48YC8",
            "sysobjectid": ".1.3.6.1.4.1.30065.1",
            "sysservices": 6,
            "description": "Arista Networks EOS version 4.28.3M",
            "ports": 48,
            "stacking": False,
            "poe": False,
            "uplinks": ["Ethernet49/1", "Ethernet50/1", "Ethernet51/1", "Ethernet52/1"],
        },
    },
    "dell": {
        "n3248te-on": {
            "model": "N3248TE-ON",
            "sysobjectid": ".1.3.6.1.4.1.674.11000.5000.100.2.1",
            "sysservices": 6,
            "description": "Dell EMC Networking N3248TE-ON, OS10 Enterprise 10.5.3.4",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": [
                "ethernet1/1/49",
                "ethernet1/1/50",
                "ethernet1/1/51",
                "ethernet1/1/52",
            ],
        },
        "s5248f-on": {
            "model": "S5248F-ON",
            "sysobjectid": ".1.3.6.1.4.1.674.11000.5000.100.2.1",
            "sysservices": 6,
            "description": "Dell EMC Networking S5248F-ON, OS10 Enterprise 10.5.3.4",
            "ports": 48,
            "stacking": True,
            "poe": False,
            "uplinks": [
                "ethernet1/1/49",
                "ethernet1/1/50",
                "ethernet1/1/51",
                "ethernet1/1/52",
            ],
        },
    },
    "fortinet": {
        "fs-448e-fpoe": {
            "model": "FS-448E-FPOE",
            "sysobjectid": ".1.3.6.1.4.1.12356",
            "sysservices": 6,
            "description": "FortiSwitch-448E-FPOE v7.2.3,build0517,221201 (GA)",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["port49", "port50", "port51", "port52"],
        },
        "fs-548d-fpoe": {
            "model": "FS-548D-FPOE",
            "sysobjectid": ".1.3.6.1.4.1.12356",
            "sysservices": 6,
            "description": "FortiSwitch-548D-FPOE v7.2.3,build0517,221201 (GA)",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["port49", "port50", "port51", "port52"],
        },
    },
    "meraki": {
        "ms390-48uxb": {
            "model": "MS390-48UXB",
            "sysobjectid": ".1.3.6.1.4.1.29671.2",
            "sysservices": 6,
            "description": "Cisco Meraki MS390-48UXB Cloud-Managed Aggregation Switch, firmware 15.21",
            "ports": 48,
            "stacking": True,
            "poe": True,
            "uplinks": ["Port49", "Port50", "Port51", "Port52"],
        },
        "ms425-48": {
            "model": "MS425-48",
            "sysobjectid": ".1.3.6.1.4.1.29671.2",
            "sysservices": 6,
            "description": "Cisco Meraki MS425-48 Cloud-Managed Aggregation Switch, firmware 15.21",
            "ports": 48,
            "stacking": True,
            "poe": False,
            "uplinks": ["Port49", "Port50", "Port51", "Port52"],
        },
    },
}


def generate_system_mib(device_info, hostname="niac-device-01"):
    """Generate System MIB (.1.3.6.1.2.1.1.x) entries."""
    lines = []
    lines.append(f".1.3.6.1.2.1.1.1.0 = STRING: {device_info['description']}")
    # Subscripted, not .get(): a template with no sysObjectID must stop the
    # generator. Defaulting these two is how nine walks came to describe an
    # Aruba, an Arista and a Palo Alto firewall while advertising a Cisco
    # switch OID -- sysObjectID is what an analyser keys vendor on, so the
    # walks read correct until something machine-read them (#2154).
    lines.append(f".1.3.6.1.2.1.1.2.0 = OID: {device_info['sysobjectid']}")
    lines.append(".1.3.6.1.2.1.1.3.0 = Timeticks: (123456789) 14 days, 6:56:07.89")
    lines.append(".1.3.6.1.2.1.1.4.0 = STRING: Network Administrator")
    lines.append(f".1.3.6.1.2.1.1.5.0 = STRING: {hostname}")
    lines.append(".1.3.6.1.2.1.1.6.0 = STRING: NiAC-Go Simulated Device")
    lines.append(f".1.3.6.1.2.1.1.7.0 = INTEGER: {device_info['sysservices']}")
    return lines


def generate_interface_mib(device_info):
    """Generate Interface MIB entries."""
    lines = []
    num_ports = device_info["ports"]

    # ifNumber
    total_ifs = num_ports + len(device_info.get("uplinks", [])) + 1  # +1 for management
    lines.append(f".1.3.6.1.2.1.2.1.0 = INTEGER: {total_ifs}")

    # Interface table entries
    for i in range(1, total_ifs + 1):
        lines.append(f".1.3.6.1.2.1.2.2.1.1.{i} = INTEGER: {i}")  # ifIndex
        if i <= num_ports:
            lines.append(
                f".1.3.6.1.2.1.2.2.1.2.{i} = STRING: GigabitEthernet1/0/{i}"
            )  # ifDescr
            lines.append(
                f".1.3.6.1.2.1.2.2.1.3.{i} = INTEGER: 6"
            )  # ifType (ethernetCsmacd)
            lines.append(
                f".1.3.6.1.2.1.2.2.1.5.{i} = Gauge32: 1000000000"
            )  # ifSpeed (1Gbps)
            lines.append(f".1.3.6.1.2.1.2.2.1.8.{i} = INTEGER: 1")  # ifOperStatus (up)
        elif i <= num_ports + len(device_info.get("uplinks", [])):
            uplink_idx = i - num_ports - 1
            lines.append(
                f".1.3.6.1.2.1.2.2.1.2.{i} = STRING: {device_info['uplinks'][uplink_idx]}"
            )
            lines.append(f".1.3.6.1.2.1.2.2.1.3.{i} = INTEGER: 6")
            lines.append(f".1.3.6.1.2.1.2.2.1.5.{i} = Gauge32: 10000000000")  # 10Gbps
            lines.append(f".1.3.6.1.2.1.2.2.1.8.{i} = INTEGER: 1")
        else:
            lines.append(f".1.3.6.1.2.1.2.2.1.2.{i} = STRING: Vlan1")  # Management
            lines.append(f".1.3.6.1.2.1.2.2.1.3.{i} = INTEGER: 53")  # propVirtual
            lines.append(f".1.3.6.1.2.1.2.2.1.5.{i} = Gauge32: 1000000000")
            lines.append(f".1.3.6.1.2.1.2.2.1.8.{i} = INTEGER: 1")

    return lines


def generate_cisco_enterprise_mib(device_info):
    """Generate Cisco-specific enterprise MIBs."""
    lines = []
    # Cisco enterprise OID prefix: .1.3.6.1.4.1.9
    lines.append(
        f".1.3.6.1.4.1.9.9.13.1.3.1.3.1 = STRING: {device_info['model']}"
    )  # entPhysicalModel
    lines.append(".1.3.6.1.4.1.9.9.13.1.3.1.5.1 = STRING: Chassis")  # entPhysicalDescr
    lines.append(
        ".1.3.6.1.4.1.9.9.13.1.3.1.11.1 = STRING: FOCCXXXXXXX"
    )  # entPhysicalSerialNum

    # CPU statistics
    lines.append(
        ".1.3.6.1.4.1.9.9.109.1.1.1.1.3.1 = Gauge32: 5"
    )  # cpmCPUTotal5sec (5%)
    lines.append(".1.3.6.1.4.1.9.9.109.1.1.1.1.4.1 = Gauge32: 8")  # cpmCPUTotal1min
    lines.append(".1.3.6.1.4.1.9.9.109.1.1.1.1.5.1 = Gauge32: 10")  # cpmCPUTotal5min

    # Memory statistics
    lines.append(
        ".1.3.6.1.4.1.9.9.48.1.1.1.5.1 = INTEGER: 4194304000"
    )  # ciscoMemoryPoolUsed (4GB)
    lines.append(
        ".1.3.6.1.4.1.9.9.48.1.1.1.6.1 = INTEGER: 12582912000"
    )  # ciscoMemoryPoolFree (12GB)

    return lines


def generate_juniper_enterprise_mib(device_info):
    """Generate Juniper-specific enterprise MIBs."""
    lines = []
    # Juniper enterprise OID prefix: .1.3.6.1.4.1.2636
    lines.append(f".1.3.6.1.4.1.2636.3.1.2.0 = STRING: {device_info['model']}")
    lines.append(".1.3.6.1.4.1.2636.3.1.3.0 = STRING: SNXXXXXXXX")  # Serial number
    return lines


def generate_walk_file(vendor, model, output_file, hostname="niac-device-01"):
    """Generate a complete walk file for the specified device."""
    if vendor not in DEVICE_TEMPLATES:
        print(f"Error: Unknown vendor '{vendor}'")
        print(f"Available vendors: {', '.join(DEVICE_TEMPLATES.keys())}")
        return False

    if model not in DEVICE_TEMPLATES[vendor]:
        print(f"Error: Unknown model '{model}' for vendor '{vendor}'")
        print(f"Available models: {', '.join(DEVICE_TEMPLATES[vendor].keys())}")
        return False

    device_info = DEVICE_TEMPLATES[vendor][model]

    print(f"Generating walk file for {vendor} {model}...")
    print(f"  Model: {device_info['model']}")
    print(f"  Ports: {device_info['ports']}")
    print(f"  Stacking: {device_info['stacking']}")
    print(f"  PoE: {device_info['poe']}")

    lines = []

    # Header comment
    # Source first, matching the shipped corpus, so regenerating a starter
    # walk produces a diff of only what changed.
    lines.append("# Source: generated")
    lines.append(f"# SNMP Walk File for {vendor.title()} {device_info['model']}")
    lines.append("# Generated by NiAC-Go walk file generator")
    lines.append(f"# Date: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    lines.append(f"# Hostname: {hostname}")
    lines.append("#")

    # Generate MIB sections
    lines.extend(generate_system_mib(device_info, hostname))
    lines.append("")
    lines.extend(generate_interface_mib(device_info))
    lines.append("")

    # Vendor-specific MIBs
    if vendor == "cisco":
        lines.extend(generate_cisco_enterprise_mib(device_info))
    elif vendor == "juniper":
        lines.extend(generate_juniper_enterprise_mib(device_info))

    # Write to file
    with open(output_file, "w") as f:
        f.write("\n".join(lines))
        f.write("\n")

    print(f"\n✅ Walk file generated: {output_file}")
    print(
        f"   Total OIDs: {len([line for line in lines if line and not line.startswith('#')])}"
    )
    return True


def list_devices():
    """List all available device templates."""
    print("\n=== Available Device Templates ===\n")
    for vendor, models in sorted(DEVICE_TEMPLATES.items()):
        print(f"{vendor.upper()}:")
        for model, info in sorted(models.items()):
            print(f"  {model:20} - {info['model']} ({info['ports']} ports)")
        print()


def main():
    parser = argparse.ArgumentParser(
        description="Generate synthetic SNMP walk files for modern network devices",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # List available devices
  python3 generate_modern_walk.py --list

  # Generate Cisco Catalyst 3850-48P walk file
  python3 generate_modern_walk.py --vendor cisco --model c3850-48p --output c3850.walk

  # Generate Juniper EX4300 walk file
  python3 generate_modern_walk.py --vendor juniper --model ex4300-48p --output ex4300.walk

  # Generate Aruba CX 6300 walk file
  python3 generate_modern_walk.py --vendor aruba --model cx6300-48g --output cx6300.walk
        """,
    )

    parser.add_argument(
        "--list", action="store_true", help="List all available device templates"
    )
    parser.add_argument(
        "--vendor",
        type=str,
        help="Device vendor (cisco, juniper, aruba, extreme, paloalto)",
    )
    parser.add_argument(
        "--model", type=str, help="Device model (use --list to see available models)"
    )
    parser.add_argument("--output", type=str, help="Output walk file path")
    parser.add_argument(
        "--hostname",
        type=str,
        default="niac-device-01",
        help="Device hostname (default: niac-device-01)",
    )

    args = parser.parse_args()

    if args.list:
        list_devices()
        return 0

    if not args.vendor or not args.model or not args.output:
        parser.print_help()
        return 1

    success = generate_walk_file(args.vendor, args.model, args.output, args.hostname)
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())
