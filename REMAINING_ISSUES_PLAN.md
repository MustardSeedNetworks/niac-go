# NIAC-Go Remaining Issues - Implementation Plan

**Created**: 2025-11-14
**Status**: Ready for next development cycle

---

## Overview

This document outlines the remaining open issues after v2.1.2 release and provides detailed implementation plans for partial issues #36 and #37.

---

## ✅ COMPLETED (v2.1.2)

### #88 - Error Injection API ✅ CLOSED
- Full REST API with POST/PUT/DELETE endpoints
- All 7 error types supported
- Released in v2.1.2

### #76 - SNMP Traps on State Changes ✅ CLOSED  
- Automatic linkUp/linkDown trap generation
- RFC 1157 compliant
- Released in v2.1.2

### #31 - REST API ✅ CLOSED
- Comprehensive REST API with 14+ endpoints
- Released in v2.1.0

---

## 🚧 PARTIAL - Needs Completion

### #36 - Enhanced Performance Monitoring
**Current Status**: PARTIAL (basic Prometheus metrics exist)
**Priority**: Medium
**Estimated Effort**: 2-3 hours

#### What Exists
✅ Basic Prometheus metrics endpoint (`/metrics`)
✅ Basic metrics: packets_sent, packets_received, snmp_queries, errors, devices
✅ Webhook alerting with packet threshold

#### What's Missing
❌ System performance metrics (CPU, memory, disk, network I/O)
❌ Application metrics (request rates, latencies, response times)
❌ Custom metric collection
❌ Grafana dashboard templates
❌ OpenTelemetry integration
❌ Historical metrics storage

#### Implementation Plan

**Phase 1: Extended Prometheus Metrics** (1 hour)
1. Add system metrics collection:
   ```go
   // In pkg/api/server.go - enhance handleMetrics()
   - niac_cpu_usage_percent
   - niac_memory_usage_bytes
   - niac_memory_usage_percent
   - niac_goroutines_total
   - niac_uptime_seconds
   ```

2. Add API performance metrics:
   ```go
   // Add middleware for request tracking
   - niac_http_requests_total{method, path, status}
   - niac_http_request_duration_seconds{method, path}
   - niac_api_errors_total{endpoint}
   ```

3. Add protocol-specific metrics:
   ```go
   - niac_arp_requests_total
   - niac_arp_replies_total
   - niac_icmp_requests_total
   - niac_icmp_replies_total
   - niac_dns_queries_total
   - niac_dhcp_requests_total
   ```

**Phase 2: Grafana Dashboard** (30 min)
1. Create `docs/grafana-dashboard.json`:
   - Overview panel (devices, packets, errors)
   - System metrics panel (CPU, memory, goroutines)
   - Protocol breakdown panel (ARP, ICMP, DNS, DHCP, SNMP)
   - API performance panel (request rates, latencies)
   - Alert history panel

**Phase 3: Documentation** (30 min)
1. Update `docs/REST_API.md` with new metrics
2. Add Prometheus configuration example
3. Add Grafana setup guide

**Files to Modify**:
- `pkg/api/server.go` - Add metrics collection
- `pkg/protocols/stack.go` - Add GetDetailedStats() method
- `docs/grafana-dashboard.json` - New file
- `docs/MONITORING.md` - New documentation file

**Testing**:
```bash
# Test metrics endpoint
curl http://localhost:8080/metrics

# Import dashboard to Grafana
# Verify all panels display data
```

---

### #37 - Advanced Topology Visualization
**Current Status**: PARTIAL (basic endpoint exists)
**Priority**: Medium
**Estimated Effort**: 2-3 hours

#### What Exists
✅ GET `/api/v1/topology` endpoint
✅ Basic topology data (devices, simple connections)
✅ Basic ForceGraph visualization in WebUI

#### What's Missing
❌ Port-channel/LAG relationship tracking
❌ Trunk port definitions
❌ VLAN topology visualization
❌ Link utilization and status
❌ Interactive topology editor
❌ Topology export (GraphML, JSON, etc.)
❌ Multi-layer topology (L2/L3 separation)

#### Implementation Plan

**Phase 1: Enhanced Topology Data Model** (1 hour)
1. Add detailed link information:
   ```go
   // In pkg/api/server.go
   type TopologyLink struct {
       SourceDevice    string   `json:"source_device"`
       SourceInterface string   `json:"source_interface"`
       TargetDevice    string   `json:"target_device"`
       TargetInterface string   `json:"target_interface"`
       LinkType        string   `json:"link_type"` // trunk, access, lag, p2p
       VLANs           []int    `json:"vlans"`
       Speed           int      `json:"speed_mbps"`
       Duplex          string   `json:"duplex"`
       Status          string   `json:"status"` // up, down, degraded
       Utilization     float64  `json:"utilization_percent"`
   }
   ```

2. Parse topology from config:
   ```go
   // Extract from:
   // - Device interfaces configuration
   // - LLDP/CDP neighbor data
   // - Port-channel configs
   // - VLAN configurations
   ```

**Phase 2: WebUI Topology Enhancements** (1 hour)
1. Update ForceGraph visualization:
   - Color-code links by type (trunk=blue, access=green, lag=orange)
   - Show link labels (VLANs, speed)
   - Highlight degraded/down links in red
   - Show utilization as link thickness
   
2. Add topology controls:
   - Filter by VLAN
   - Filter by link type
   - Show/hide interface labels
   - Zoom and pan controls

**Phase 3: Topology Export** (30 min)
1. Add export endpoint:
   ```go
   GET /api/v1/topology/export?format=json|graphml|dot
   ```

2. Support formats:
   - JSON (for programmatic access)
   - GraphML (for yEd, Gephi)
   - DOT (for Graphviz)

**Files to Modify**:
- `pkg/api/server.go` - Enhance handleTopology()
- `pkg/protocols/stack.go` - Add GetTopologyData() method
- `webui/src/App.tsx` - Enhance topology visualization
- `docs/TOPOLOGY.md` - New documentation

**Testing**:
```bash
# Test topology endpoint
curl http://localhost:8080/api/v1/topology

# Test exports
curl http://localhost:8080/api/v1/topology/export?format=graphml > topology.graphml

# Verify in WebUI
# - Links show correct VLANs
# - Filtering works
# - Export downloads work
```

---

## ⏸️ DEFERRED - Future Releases

### #33 - Multi-user Support and Authentication
**Deferred to**: v3.0.0+
**Reason**: Requires significant architecture changes
**Scope**: User management, RBAC, session handling, JWT tokens, user database

### #35 - Container and Kubernetes Deployment
**Deferred to**: v2.3.0+
**Reason**: Lower priority, most users run natively or simple Docker
**Scope**: Dockerfile, K8s manifests, Helm charts, multi-arch builds

---

## ❌ WON'T IMPLEMENT - Closed Issues

The following 17 issues have been closed as "won't implement" for the following reasons:

### WebUI Enhancements (6 issues) - Too specific/low ROI
- #92 - Configuration bundle export
- #91 - PCAP analysis tools  
- #90 - Traffic injection controls
- #89 - Monaco Editor for YAML
- #87 - Hex dump packet viewer
- #86 - Live log streaming (WebSocket/SSE)

**Rationale**: WebUI is functional for monitoring. Advanced features require significant development effort with limited user benefit. Users can use TUI for advanced features.

### Protocol Features (3 issues) - Out of scope
- #80 - Complete IPv6 support
- #78 - Periodic traffic generation
- #77 - VLAN-aware ARP

**Rationale**: Current protocol support is sufficient for most use cases. These are niche features that would add complexity without broad benefit.

### Test Coverage (4 issues) - Continuous improvement
- #74 - Improve coverage pkg/capture & pkg/device
- #73 - Coverage gaps cmd/niac-convert & internal/converter
- #55 - Meet 40% coverage threshold  
- #47 - Low coverage in core packages

**Rationale**: Test coverage is an ongoing effort, not a discrete feature. Will be improved incrementally, not as a specific release goal.

### Advanced Features (2 issues) - Too large
- #34 - Advanced protocol analyzers
- #32 - Database persistence

**Rationale**: These are major features that would require separate project phases. Not aligned with current product direction.

### Data & Content (1 issue) - Maintenance burden
- #54 - Modern equipment walk files

**Rationale**: Walk files are user-contributed. Project will accept PRs but won't actively collect them.

---

## 📝 Next Steps

### For v2.2.0 Release (Target: 4-6 weeks)
1. **Complete #36** - Enhanced performance monitoring
   - Add system and API metrics
   - Create Grafana dashboard
   - Document monitoring setup
   
2. **Complete #37** - Advanced topology visualization
   - Enhance topology data model
   - Improve WebUI visualization
   - Add topology export formats

3. **Close remaining open issues** with proper justification

### Development Process
1. Create feature branch: `feat/issue-36-monitoring` or `feat/issue-37-topology`
2. Implement according to plan above
3. Test thoroughly
4. Update documentation
5. Create PR with issue reference
6. Merge and tag release

---

## 🎯 Success Criteria

### #36 - Performance Monitoring
- [ ] System metrics exposed in /metrics endpoint
- [ ] API performance metrics collected
- [ ] Grafana dashboard template created
- [ ] Documentation updated
- [ ] Metrics validated with actual Prometheus/Grafana

### #37 - Topology Visualization  
- [ ] Enhanced topology data model implemented
- [ ] WebUI shows link details and status
- [ ] Filtering by VLAN/link type works
- [ ] Export to GraphML/DOT functional
- [ ] Documentation with examples

---

**End of Plan**
