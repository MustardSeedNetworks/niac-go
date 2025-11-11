# Modern Walk Files Strategy

Implementation plan for generating and obtaining SNMP walk files for modern network equipment.

**GitHub Issue:** #54
**Status:** In Progress
**Last Updated:** 2025-01-11

## Table of Contents

- [Overview](#overview)
- [Approach 1: Synthetic Generation](#approach-1-synthetic-generation)
- [Approach 2: Virtual Lab Capture](#approach-2-virtual-lab-capture)
- [Approach 3: Community Contributions](#approach-3-community-contributions)
- [Approach 4: Vendor Partnerships](#approach-4-vendor-partnerships)
- [Priority Devices](#priority-devices)
- [Implementation Phases](#implementation-phases)
- [Tools and Scripts](#tools-and-scripts)

## Overview

The goal is to add SNMP walk files for modern network equipment to enable realistic simulation of current-generation devices. This addresses the gap where existing walk files are primarily from older switch models.

### Current State

- **Total Walk Files:** 555 sanitized files
- **Vendors:** 17 vendors
- **Primary Gap:** Modern Cisco (3650/3850/9xxx), Juniper (EX/QFX), Aruba (CX), and other current-generation equipment

### Target Devices

**Cisco:**
- Catalyst 3650 Series (stackable access)
- Catalyst 3850 Series (stackable access/distribution)
- Catalyst 9200/9300/9400/9500 (next-gen campus)
- Nexus 9300/9500 (ACI-ready data center)

**Juniper:**
- EX2300/EX3400/EX4300 (access)
- QFX5100/QFX5200 (data center)
- MX-Series routers

**Aruba:**
- CX 6200/6300/6400 (campus access/aggregation)
- CX 8320/8360/9300 (data center)

**Other:**
- Fortinet FortiGate/FortiSwitch
- Extreme X-Series (X435, X465, X690, X870)
- Palo Alto PA-Series firewalls

## Approach 1: Synthetic Generation

**Status:** ✅ Implemented

Generate synthetic walk files based on vendor MIB documentation and device specifications.

### Advantages

- ✅ Fast - no hardware required
- ✅ Customizable - any device model
- ✅ Consistent - deterministic output
- ✅ No licensing concerns
- ✅ Already sanitized

### Limitations

- ❌ May miss vendor-specific quirks
- ❌ Not from real hardware
- ❌ Limited to documented MIBs
- ❌ May not match exact firmware behavior

### Implementation

**Tool Created:** `scripts/generate_modern_walk.py`

**Features:**
- Template-based generation
- Support for multiple vendors
- Configurable device models
- Realistic System, Interface, and Enterprise MIBs
- Standard SNMP walk format output

**Supported Devices (v1.0):**

| Vendor | Model | Ports | Features |
|--------|-------|-------|----------|
| Cisco | C3850-48P | 48 | Stackable, PoE, 4x10G uplinks |
| Cisco | C3650-48P | 48 | Stackable, PoE, 2x10G uplinks |
| Cisco | C9300-48P | 48 | Next-gen, stackable, PoE |
| Cisco | N9K-C9300 | 32 | Data center, ACI-ready |
| Juniper | EX4300-48P | 48 | Stackable, PoE |
| Juniper | QFX5100-48S | 48 | Data center, 40/100G capable |
| Aruba | CX6300-48G | 48 | Campus, stackable |
| Aruba | CX8360-48Y8C | 48 | Data center, 25G capable |
| Extreme | X465-48W | 48 | Stackable, PoE |
| Palo Alto | PA-440 | 8 | Next-gen firewall |

**Usage:**

```bash
# List available devices
./scripts/generate_modern_walk.py --list

# Generate Cisco Catalyst 3850
./scripts/generate_modern_walk.py \
  --vendor cisco \
  --model c3850-48p \
  --output examples/device_walks_sanitized/cisco/niac-cisco-c3850-48p.walk \
  --hostname niac-core-sw-01

# Generate all modern devices
for model in c3850-48p c3650-48p c9300-48p; do
  ./scripts/generate_modern_walk.py \
    --vendor cisco \
    --model $model \
    --output examples/device_walks_sanitized/cisco/niac-cisco-$model.walk
done
```

**Generated Walk Files Include:**
- System MIB (.1.3.6.1.2.1.1.*) - sysDescr, sysObjectID, sysUpTime, sysName, etc.
- Interface MIB (.1.3.6.1.2.1.2.*) - ifDescr, ifType, ifSpeed, ifOperStatus
- Interface Extensions (.1.3.6.1.2.1.31.*) - ifName, ifHCInOctets, ifHighSpeed
- Vendor Enterprise MIBs (Cisco .1.3.6.1.4.1.9.*, Juniper .1.3.6.1.4.1.2636.*)

### Expanding Templates

To add new devices, edit `scripts/generate_modern_walk.py`:

```python
DEVICE_TEMPLATES = {
    'cisco': {
        'c9400-48p': {  # New device
            'model': 'C9400-48P',
            'description': 'Cisco IOS Software [Gibraltar], Catalyst L3 Switch Software (CAT9K_IOSXE), Version 17.6.3',
            'ports': 48,
            'stacking': False,
            'poe': True,
            'uplinks': ['FortyGigabitEthernet1/1/1', 'FortyGigabitEthernet1/1/2'],
        },
    },
}
```

## Approach 2: Virtual Lab Capture

**Status:** 📋 Planned

Capture SNMP walks from vendor virtual appliances and simulators.

### Advantages

- ✅ Real device behavior
- ✅ Accurate firmware responses
- ✅ Complete MIB coverage
- ✅ Vendor-specific quirks preserved

### Limitations

- ❌ Requires virtual lab access
- ❌ May need vendor licensing
- ❌ Time-consuming setup
- ❌ Needs sanitization

### Virtual Lab Options

#### Cisco

**Cisco Modeling Labs (CML) / VIRL:**
- Catalyst 9000v (IOS-XE virtual switch)
- Nexus 9000v (NX-OS virtual switch)
- CSR1000v (IOS-XE virtual router)

**Cisco DevNet Sandbox:**
- Free access to virtual devices
- Pre-configured labs
- API access for automation

**Setup:**
```bash
# 1. Deploy virtual device in CML
# 2. Configure SNMP
(config)# snmp-server community public RO
(config)# snmp-server contact admin@example.com
(config)# snmp-server location NiAC-Go Virtual Lab

# 3. Capture walk
snmpwalk -v2c -c public <device_ip> .1 > cisco-c9300v.walk

# 4. Sanitize
./niac sanitize --input cisco-c9300v.walk --output niac-cisco-c9300v.walk
```

#### Juniper

**Juniper vLabs:**
- vMX (virtual MX router)
- vSRX (virtual SRX firewall)
- vQFX (virtual QFX switch)

**Juniper NITA (Network Integration Test Automation):**
- Vagrant-based virtual lab
- Ansible automation
- Free and open-source

**Setup:**
```bash
# 1. Deploy vQFX in vLabs
# 2. Configure SNMP
set snmp community public authorization read-only
set snmp contact admin@example.com
set snmp location "NiAC-Go Virtual Lab"

# 3. Capture walk
snmpwalk -v2c -c public <device_ip> .1 > juniper-vqfx.walk
```

#### Aruba

**Aruba CX Simulation Software (CX-SSW):**
- OVA virtual appliance
- Full CX operating system
- Free for labs

**Setup:**
```bash
# 1. Import CX-SSW OVA into VMware/VirtualBox
# 2. Configure SNMP
(config)# snmpv2-server community public type ro
(config)# snmp-server contact admin@example.com
(config)# snmp-server location "NiAC-Go Virtual Lab"

# 3. Capture walk
snmpwalk -v2c -c public <device_ip> .1 > aruba-cx6300v.walk
```

#### GNS3/EVE-NG

**GNS3 (Graphical Network Simulator):**
- Supports Cisco, Juniper, Aruba images
- Free and open-source
- Large community

**EVE-NG (Emulated Virtual Environment - Next Generation):**
- Commercial and community editions
- Supports many vendor images
- Web-based interface

### Capture Automation Script

```bash
#!/bin/bash
# capture_virtual_walk.sh - Automate virtual lab walk capture

DEVICE_IP=$1
COMMUNITY=${2:-public}
DEVICE_NAME=$3

echo "Capturing walk from $DEVICE_IP..."

# Full walk
snmpwalk -v2c -c $COMMUNITY $DEVICE_IP .1 > "${DEVICE_NAME}_raw.walk"

# Sanitize
../niac sanitize \
  --input "${DEVICE_NAME}_raw.walk" \
  --output "../examples/device_walks_sanitized/cisco/niac-${DEVICE_NAME}.walk"

echo "Walk file created: niac-${DEVICE_NAME}.walk"
```

## Approach 3: Community Contributions

**Status:** 📋 Planned

Accept walk file contributions from the community.

### Advantages

- ✅ Real hardware captures
- ✅ Production firmware versions
- ✅ Diverse models and configs
- ✅ Community engagement

### Limitations

- ❌ Quality control needed
- ❌ Sanitization required
- ❌ Inconsistent documentation
- ❌ Potential licensing issues

### Contribution Process

**See:** [WALK_FILES.md - Contributing Section](WALK_FILES.md#contributing-walk-files)

1. **Capture walk file** from device (with permission)
2. **Sanitize** using NiAC-Go tools
3. **Document** device details
4. **Test** with NiAC-Go
5. **Submit PR** with walk file

**Contribution Template:**

```markdown
## Walk File Contribution: Cisco Catalyst 9300-48P

**Device Information:**
- Vendor: Cisco
- Model: C9300-48P
- Software Version: IOS XE 17.6.3
- Capture Date: 2025-01-11
- Source: Physical hardware in production

**Walk File Details:**
- Filename: niac-cisco-c9300-48p.walk
- Size: 2.5 MB
- OID Count: ~15,000
- Sanitized: Yes

**Testing:**
- ✅ Validates with niac validate
- ✅ Responds to snmpwalk queries
- ✅ System MIB complete
- ✅ Interface MIB complete

**Additional Notes:**
- Includes StackWise-480 stacking MIBs
- PoE status and statistics included
- UADP (Unified Access Data Plane) metrics present
```

### Community Outreach

**Channels:**
- GitHub Discussions
- Reddit (r/networking, r/Cisco, r/networking)
- Network Engineering forums
- Vendor community sites
- Conference presentations

**Incentives:**
- Contributor recognition in README
- Walk file attribution
- Priority feature requests
- Project collaboration

## Approach 4: Vendor Partnerships

**Status:** 📋 Future

Partner with vendors for official walk files.

### Advantages

- ✅ Officially supported
- ✅ Accurate and complete
- ✅ Regular updates
- ✅ Legal clarity

### Limitations

- ❌ Time-consuming
- ❌ May require NDAs
- ❌ Limited vendor interest
- ❌ Bureaucratic

### Potential Partners

**Cisco DevNet:**
- Developer program
- API and tool support
- Community engagement

**Juniper Open Learning:**
- Educational resources
- Lab access
- Community support

**Aruba Airheads Community:**
- Developer community
- Virtual lab access
- Technical resources

### Outreach Template

```
Subject: Partnership Opportunity - SNMP Walk Files for Network Simulation

Dear [Vendor] Developer Relations,

NiAC-Go is an open-source network simulation tool used for testing, training, and development. We're seeking to expand our library of SNMP walk files to include modern [Vendor] equipment.

Benefits to [Vendor]:
- Increased developer adoption
- Training and education use
- Testing tool integration
- Community engagement

We would appreciate:
- Official SNMP walk files for key models
- Permission to distribute walk files
- Technical review of our implementations

Project: https://github.com/krisarmstrong/niac-go
License: [License]
Users: [Statistics]

Would you be interested in discussing a partnership?

Best regards,
[Name]
```

## Priority Devices

Based on market adoption and user demand:

### Phase 1: High Priority (Q1 2025)

**Cisco (Synthetic Generation - ✅ Complete):**
- ✅ Catalyst 3850-48P
- ✅ Catalyst 3650-48P
- ✅ Catalyst 9300-48P
- ✅ Nexus 9300

**Next Steps:**
- Catalyst 9200/9400/9500 templates
- Nexus 9500 template
- Testing and validation

### Phase 2: Medium Priority (Q2 2025)

**Juniper:**
- ✅ EX4300-48P (synthetic)
- ✅ QFX5100-48S (synthetic)
- TODO: Virtual lab captures (vQFX)

**Aruba:**
- ✅ CX 6300-48G (synthetic)
- ✅ CX 8360-48Y8C (synthetic)
- TODO: CX Simulation Software captures

**Extreme:**
- ✅ X465-48W (synthetic)
- TODO: X690/X870 templates

### Phase 3: Lower Priority (Q3 2025)

**Palo Alto:**
- ✅ PA-440 (synthetic)
- TODO: PA-3200/PA-5200 templates

**Fortinet:**
- TODO: FortiGate 200F/400F
- TODO: FortiSwitch 448E/524D

## Implementation Phases

### Phase 1: Synthetic Generation (Current)

**Timeline:** ✅ Complete (January 2025)

**Deliverables:**
- ✅ Walk file generator tool
- ✅ 10 device templates
- ✅ Generated walk files
- ✅ Documentation

**Status:** Complete

### Phase 2: Validation and Enhancement (Q1 2025)

**Timeline:** January - March 2025

**Tasks:**
- [ ] Test generated walk files with NiAC-Go
- [ ] Compare with real device output
- [ ] Enhance templates with additional MIBs
- [ ] Add more device models (20+ total)
- [ ] Community feedback and iteration

**Success Criteria:**
- Generated walk files work with NiAC-Go
- SNMP queries return expected values
- Community validation positive
- 20+ modern devices supported

### Phase 3: Virtual Lab Captures (Q2 2025)

**Timeline:** April - June 2025

**Tasks:**
- [ ] Set up Cisco CML/DevNet Sandbox
- [ ] Set up Juniper vLabs
- [ ] Set up Aruba CX-SSW
- [ ] Capture walks from virtual devices
- [ ] Compare synthetic vs virtual captures
- [ ] Document differences

**Success Criteria:**
- Virtual lab environments operational
- 10+ virtual device captures
- Comparison report published
- Best practices documented

### Phase 4: Community Contributions (Q3 2025)

**Timeline:** July - September 2025

**Tasks:**
- [ ] Publish contribution guidelines
- [ ] Create submission templates
- [ ] Set up review process
- [ ] Promote in communities
- [ ] Accept and merge contributions

**Success Criteria:**
- 10+ community contributions
- Quality review process established
- Active contributor community
- Regular submissions

## Tools and Scripts

### Walk File Generator

**Location:** `scripts/generate_modern_walk.py`

**Features:**
- Template-based generation
- Multiple vendors supported
- Configurable output
- CLI interface

**Usage:**
```bash
# List devices
./scripts/generate_modern_walk.py --list

# Generate walk file
./scripts/generate_modern_walk.py \
  --vendor cisco \
  --model c3850-48p \
  --output walk.snmp \
  --hostname my-device

# Batch generation
for model in c3850-48p c9300-48p; do
  ./scripts/generate_modern_walk.py --vendor cisco --model $model --output cisco-$model.walk
done
```

### Walk File Analyzer

**Location:** `/tmp/analyze_walk.py` (needs integration)

**Purpose:** Analyze existing walk files to extract patterns for template creation.

**Usage:**
```bash
python3 analyze_walk.py examples/device_walks_sanitized/cisco/niac-cisco-c2960-70.walk
```

### Sanitization Tool

**Built into NiAC-Go:**

```bash
# Single file
niac sanitize --input raw.walk --output sanitized.walk

# Batch processing
niac sanitize --batch \
  --input-dir raw_walks/ \
  --output-dir sanitized_walks/ \
  --mapping-file mapping.json
```

## Success Metrics

### Quantitative

- **Walk Files Added:** Target 50+ modern devices by end of 2025
- **Vendor Coverage:** All 6 target vendors represented
- **Community Contributions:** 10+ contributions
- **Usage:** Walk files used in 100+ NiAC-Go deployments

### Qualitative

- Walk files validated against real devices
- Community feedback positive
- Documentation comprehensive
- Easy contribution process

## Next Steps

1. ✅ **Complete Phase 1** - Synthetic generation tool
2. **Test generated walk files** with NiAC-Go configurations
3. **Document differences** between synthetic and real captures
4. **Expand templates** to 20+ devices
5. **Set up virtual labs** for Phase 3
6. **Launch community contribution** program

## References

- [WALK_FILES.md](WALK_FILES.md) - Walk file documentation
- [GitHub Issue #54](https://github.com/krisarmstrong/niac-go/issues/54) - Original request
- `scripts/generate_modern_walk.py` - Generation tool
- `examples/device_walks_sanitized/` - Walk file repository

## Contact

For questions, suggestions, or contributions:
- **GitHub Issues:** https://github.com/krisarmstrong/niac-go/issues/54
- **Email:** [Project maintainers]
- **Community:** [Discord/Slack/Forum]

---

**Last Updated:** 2025-01-11
**Document Version:** 1.0
**Status:** Phase 1 Complete, Phase 2 In Planning
