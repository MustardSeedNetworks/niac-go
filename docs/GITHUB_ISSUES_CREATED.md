# v1.23.0 GitHub Issues Created

**Date:** January 8, 2025
**Total Issues:** 6 new issues (#48-#53)

---

## Issue Summary

### #48: Walk file sanitization tool with NiAC-Go branding
**Priority:** High
**Status:** Open
**URL:** https://github.com/krisarmstrong/niac-go/issues/48

**Scope:**
- Sanitize 555 walk files with NiAC-Go branding
- Deterministic IP mapping (10.0.0.0/8)
- Contact: `netadmin@niac-go.com`
- Device naming: `niac-core-sw-01` pattern
- JSON mapping file for consistency
- Batch processing support

---

### #49: Topology relationship tracking
**Priority:** High
**Status:** Open
**URL:** https://github.com/krisarmstrong/niac-go/issues/49

**Scope:**
- Port-channel/LAG YAML configuration
- Trunk port definitions
- Cross-device link references
- LLDP/CDP neighbor updates
- Topology validation

---

### #50: Walk file relationship analyzer tool
**Priority:** Medium (stretch goal)
**Status:** Open
**URL:** https://github.com/krisarmstrong/niac-go/issues/50

**Scope:**
- `niac analyze-walk` command
- Extract interface relationships
- Parse neighbor tables
- Generate topology YAML from walks

---

### #51: SNMP configuration examples (CRITICAL)
**Priority:** CRITICAL
**Status:** Open
**URL:** https://github.com/krisarmstrong/niac-go/issues/51

**Scope:**
- 4+ SNMP example files
- Basic agent configuration
- Trap generation examples
- Multi-device SNMP scenarios
- **NOTE:** Feature exists since v1.6.0 but has ZERO examples!

---

### #52: Topology configuration examples
**Priority:** High
**Status:** Open
**URL:** https://github.com/krisarmstrong/niac-go/issues/52

**Scope:**
- 5+ topology examples
- Two-switch trunk scenarios
- Router-Switch-AP topologies
- Port-channel labs
- VLAN configurations
- DHCP relay scenarios

**Dependency:** Requires #49

---

### #53: Protocol combination examples
**Priority:** Medium
**Status:** Open
**URL:** https://github.com/krisarmstrong/niac-go/issues/53

**Scope:**
- 4+ enterprise scenario examples
- Enterprise router (full-featured)
- Data center switch
- Access switch with PoE
- Wireless controller

---

## Implementation Priority

### Must Have (v1.23.0)
1. **#51** - SNMP examples (CRITICAL - quick win)
2. **#48** - Walk file sanitization
3. **#49** - Topology support in YAML

### Should Have (v1.23.0)
4. **#52** - Topology examples
5. **#53** - Protocol combination examples

### Nice to Have (v1.23.0)
6. **#50** - Walk analyzer tool (stretch goal)

---

## NiAC-Go Branding Guidelines

### Company Identity
- **Name:** NiAC-Go (not "NIAC-Go Corporation")
- **Domain:** niac-go.com / niac-go.local
- **Contact:** netadmin@niac-go.com

### Network Architecture
```
10.0.0.0/16  - Data Center West
10.1.0.0/16  - Data Center East  
10.2.0.0/16  - Corporate Campus
10.3.0.0/16  - Remote Offices
10.100.0.0/16 - Management Network
10.200.0.0/16 - Guest/IoT Network
```

### Device Naming
Pattern: `niac-<location>-<type>-<number>`

Examples:
- `niac-core-sw-01`
- `niac-edge-rtr-west`
- `niac-dc-srv-web01`
- `niac-ap-floor3-201`

### What We Keep
- ✅ Serial numbers
- ✅ MAC addresses
- ✅ Hardware models
- ✅ Interface configs
- ✅ VLAN IDs

### What We Transform
- 🔄 IP addresses
- 🔄 Hostnames
- 🔄 DNS domains
- 🔄 Contact info
- 🔄 Location strings
- 🔄 Community strings

---

## Timeline

**Total Duration:** 4 weeks

| Week | Focus | Issues |
|------|-------|--------|
| 1 | SNMP Examples + Sanitization Design | #51, #48 |
| 2 | Sanitization Implementation | #48 |
| 3 | Topology Support | #49, #52 |
| 4 | Examples + Polish | #52, #53, #50 |

---

## Success Criteria

- [ ] All 6 issues resolved
- [ ] 555 walk files sanitized with NiAC-Go branding
- [ ] SNMP examples created (fixing critical gap)
- [ ] Topology examples demonstrate real networks
- [ ] All examples validate successfully
- [ ] Documentation complete
- [ ] Tests pass
- [ ] Ready for v1.23.0 release

---

**Next Steps:**
1. Start with #51 (SNMP examples) - quick win, critical gap
2. Begin #48 design and implementation
3. Design #49 YAML schema
4. Implement remaining issues
5. Test, document, release v1.23.0
