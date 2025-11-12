# NiAC-Go - Branding Guide

**Purpose:** Fictional company identity for sanitized network examples
**Version:** 1.0
**Date:** January 8, 2025

---

## Company Overview

**NiAC-Go** is a fictional technology company used for all sanitized network examples and demonstrations.

**Full Name:** Network Infrastructure As Code - Go Corporation
**Ticker Symbol:** NIAC (fictional NASDAQ listing)
**Industry:** Enterprise Network Solutions
**Founded:** 2025 (in our examples)
**Headquarters:** San Francisco, CA (fictional)

---

## Network Architecture

### IP Address Allocation

```
NiAC-Go Network Map
═══════════════════════════════════════

10.0.0.0/8 - NIAC-Go Internal Network
├── 10.0.0.0/16    - Data Center West (San Francisco)
├── 10.1.0.0/16    - Data Center East (New York)
├── 10.2.0.0/16    - Corporate Campus (San Francisco)
├── 10.3.0.0/16    - Remote Offices
├── 10.100.0.0/16  - Management Network (OOB)
└── 10.200.0.0/16  - Guest/IoT Network

198.51.100.0/24 - NIAC-Go DMZ (Public-Facing Services)
203.0.113.0/24  - NIAC-Go Cloud Services

Domain Names:
├── niac-go.com      - Public internet domain
└── niac-go.local    - Internal Active Directory domain
```

### Data Center West (10.0.0.0/16)
```
10.0.0.0/24    - Core Network Infrastructure
  .1-.10       - Core routers (niac-core-rtr-01 through 10)
  .11-.30      - Core switches (niac-core-sw-01 through 20)
  .31-.50      - Distribution switches

10.0.1.0/24    - Server VLAN (Production)
  .1-.100      - Application servers
  .101-.200    - Database servers
  .201-.254    - Storage systems

10.0.2.0/24    - Server VLAN (Development)
10.0.3.0/24    - Server VLAN (QA/Test)

10.0.10.0/24   - Employee Network (Floor 1)
10.0.11.0/24   - Employee Network (Floor 2)
10.0.12.0/24   - Employee Network (Floor 3)

10.0.20.0/24   - Conference Rooms / Collaboration
10.0.30.0/24   - Wireless (Corporate)
```

### Data Center East (10.1.0.0/16)
```
10.1.0.0/24    - Core Network Infrastructure
10.1.1.0/24    - Server VLAN (Production)
10.1.2.0/24    - Server VLAN (DR/Backup)
10.1.10.0/24   - Remote Office VPN Concentrators
```

### Corporate Campus (10.2.0.0/16)
```
10.2.0.0/24    - Building A - Network Infrastructure
10.2.1.0/24    - Building A - Employees
10.2.10.0/24   - Building B - Network Infrastructure
10.2.11.0/24   - Building B - Employees
10.2.20.0/24   - Building C - R&D Lab
```

### Management Network (10.100.0.0/16)
```
10.100.0.0/24  - Network Device Management
  .1-.50       - Router management interfaces
  .51-.100     - Switch management interfaces
  .101-.150    - Wireless controller management
  .151-.200    - Security appliances

10.100.1.0/24  - Server iLO/iDRAC/IPMI
10.100.2.0/24  - Storage Management
10.100.10.0/24 - Monitoring/Logging Systems
```

---

## Device Naming Convention

### Pattern
```
niac-<location>-<type>-<identifier>
```

### Location Codes
```
core        - Core/backbone
dist        - Distribution layer
access      - Access layer
edge        - Edge/perimeter
dc-west     - Data Center West
dc-east     - Data Center East
campus      - Corporate Campus
remote      - Remote offices
floor<N>    - Building floor number
bldg<N>     - Building identifier
```

### Device Types
```
rtr         - Router
sw          - Switch
fw          - Firewall
lb          - Load balancer
ap          - Access Point
wlc         - Wireless LAN Controller
srv         - Server
nas         - Network Attached Storage
san         - Storage Area Network
```

### Examples

**Network Infrastructure:**
```
niac-core-rtr-01          Core router #1 (primary)
niac-core-rtr-02          Core router #2 (secondary)
niac-edge-rtr-west        Edge router (DC West)
niac-core-sw-01           Core switch #1
niac-dist-sw-floor3       Distribution switch, floor 3
niac-access-sw-12         Access switch #12
niac-dc-fw-01             Data center firewall
niac-campus-lb-01         Campus load balancer
```

**Wireless:**
```
niac-wlc-dc-west          Wireless controller (DC West)
niac-ap-floor1-101        Access point, floor 1, room 101
niac-ap-bldg2-conf        Access point, building 2, conference room
```

**Servers:**
```
niac-dc-srv-web01         Web server #1
niac-dc-srv-db-primary    Primary database server
niac-dc-srv-app05         Application server #5
niac-dc-nas-backup        Backup NAS system
```

---

## Contact Information

### Primary Contacts
```
Technical Contact:     netadmin@niac-go.com
Network Operations:    noc@niac-go.com
Security Team:         security@niac-go.com
Help Desk:            helpdesk@niac-go.com
Executive Contact:     cto@niac-go.com
```

### Location Information
```
Primary Format:   NiAC-Go - <Facility> - <Additional>

Examples:
- NiAC-Go - DC-WEST - Core Network
- NiAC-Go - DC-EAST - Backup Facility
- NiAC-Go - Campus - Building A Floor 3
- NiAC-Go - Remote Office - Seattle
```

### Organization Names
```
Primary:     NiAC-Go
Short:       NIAC-Go Corp
Department:  NIAC-Go IT Operations
Division:    NIAC-Go Network Engineering
```

---

## SNMP Configuration

### Community Strings
```
Read-Only:      public (examples only)
                niac-go-ro (alternative)

Read-Write:     niac-go-rw (examples only)
                private (generic examples)

Trap Community: niac-go-trap
```

**Security Note:** All examples use SNMPv2c with simple community strings for demonstration. Production deployments should use SNMPv3 with User-based Security Model (USM).

### SNMP Locations
```
sysLocation Examples:
- NiAC-Go - DC-WEST - Rack A12
- NiAC-Go - Campus Bldg A - MDF Room 101
- NiAC-Go - Remote Office Seattle
```

### SNMP Contact
```
sysContact Default:
- netadmin@niac-go.com

Alternative Contacts:
- NIAC-Go NOC <noc@niac-go.com>
- Network Operations Center - niac-go.com
```

---

## VLAN Design

### Standard VLANs
```
VLAN 1     - Native VLAN (unused)
VLAN 10    - Server Production
VLAN 11    - Server Development
VLAN 12    - Server QA/Test
VLAN 20    - Employee Workstations
VLAN 30    - Corporate Wireless
VLAN 40    - Guest Wireless
VLAN 50    - Voice/VoIP
VLAN 60    - Security Cameras / Building Automation
VLAN 70    - Printers/Copiers
VLAN 100   - Management (OOB)
VLAN 200   - IoT Devices
VLAN 999   - Quarantine/Remediation
```

---

## DNS Configuration

### Domains
```
External:  niac-go.com
Internal:  niac-go.local
```

### DNS Servers
```
Primary DNS:      10.0.0.53 (dns1.niac-go.local)
Secondary DNS:    10.1.0.53 (dns2.niac-go.local)
External DNS:     198.51.100.53 (ns1.niac-go.com)
```

### Sample DNS Records
```
# Infrastructure
niac-core-rtr-01.niac-go.local      10.0.0.1
niac-core-sw-01.niac-go.local       10.0.0.11

# Services
www.niac-go.com                      198.51.100.80
mail.niac-go.com                     198.51.100.25
vpn.niac-go.com                      198.51.100.100

# Internal Services
intranet.niac-go.local               10.0.1.100
fileserver.niac-go.local             10.0.1.50
```

---

## Sanitization Examples

### Before Sanitization
```
sysName.0 = STRING: prod-core-switch-01
sysLocation.0 = STRING: Company XYZ - Building 5 Rack A12
sysContact.0 = STRING: John Doe <jdoe@company.com>
IP-MIB::ipAdEntAddr.172.21.96.10 = IpAddress: 172.21.96.10
SNMPv2-MIB::sysDescr.0 = STRING: Cisco IOS Software
IF-MIB::ifDescr.1 = STRING: GigabitEthernet0/1
```

### After Sanitization
```
sysName.0 = STRING: niac-core-sw-01
sysLocation.0 = STRING: NiAC-Go - DC-WEST - Rack A12
sysContact.0 = STRING: netadmin@niac-go.com
IP-MIB::ipAdEntAddr.10.0.0.11 = IpAddress: 10.0.0.11
SNMPv2-MIB::sysDescr.0 = STRING: Cisco IOS Software
IF-MIB::ifDescr.1 = STRING: GigabitEthernet0/1
```

**Note:** Vendor information (Cisco IOS) and interface names are KEPT as they're not sensitive.

---

## Usage Guidelines

### When to Use NIAC-Go Corp Branding

✅ **Always Use For:**
- Example configurations
- Documentation screenshots
- Training materials
- Public demonstrations
- Conference presentations
- Blog posts and articles
- GitHub examples

❌ **Never Use For:**
- Real production networks
- Customer demonstrations (use their branding)
- Security testing on real networks
- Actual network equipment configuration

### Consistency Rules

1. **Always use the mapping file** - Ensures IP consistency across 555 walk files
2. **Maintain device roles** - Don't change a router to a switch during sanitization
3. **Preserve network topology** - Keep VLAN IDs, interface counts, connections
4. **Use realistic data** - Make it look like a real company network
5. **Document transformations** - Keep the JSON mapping file for reference

---

## Legal Notice

**NiAC-Go is a fictional company** created for educational and demonstration purposes only. Any similarity to real companies, products, or networks is purely coincidental.

All IP addresses use RFC 1918 (private) and RFC 5737 (documentation) ranges to avoid conflicts with real networks.

All examples are sanitized from real network data but do not represent any specific company or deployment.

---

**Created for:** NIAC-Go v1.23.0
**Last Updated:** January 8, 2025
**Maintained by:** NIAC-Go Development Team
