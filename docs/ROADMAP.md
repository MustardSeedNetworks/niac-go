# NIAC-Go Roadmap

## Completed Releases ✅

### v1.12.0 - Interactive TUI ✅
**Released**: January 7, 2025
- ✅ `niac interactive` command for Terminal UI mode
- ✅ Real-time device monitoring
- ✅ Interactive error injection
- ✅ Bubble Tea-based interface

### v1.11.0 - Templates & Quick Start ✅
**Released**: January 7, 2025
- ✅ 7 pre-built configuration templates
- ✅ Template commands (list, show, use)
- ✅ Embedded templates in binary

### v1.10.0 - Foundation & Validation ✅
**Released**: January 7, 2025
- ✅ Modern CLI framework (Cobra)
- ✅ `niac validate` command
- ✅ Comprehensive configuration validator
- ✅ Structured error reporting
- ✅ JSON output for CI/CD

### v1.9.0 - Code Quality ✅
**Released**: January 6, 2025
- ✅ Complexity reduction
- ✅ Integer overflow fixes
- ✅ 30+ new constants

### v1.7.0 - Testing & Quality ✅
**Released**: January 5, 2025
- ✅ 87 new tests
- ✅ 50%+ test coverage
- ✅ Configuration validator

### v1.6.0 - Traffic & SNMP Traps ✅
**Released**: January 4, 2025
- ✅ Configurable traffic patterns
- ✅ SNMP trap generation
- ✅ Complete YAML configuration

### v1.5.0 - Enhanced Configuration ✅
**Released**: January 3, 2025
- ✅ Color-coded debug output
- ✅ Per-protocol debug control
- ✅ Multiple IPs per device

### v1.0.0 - Initial Release ✅
**Released**: December 20, 2024
- ✅ Complete protocol stack (19 protocols)
- ✅ Interactive error injection
- ✅ Device simulation

---

## Completed: v1.13.0 - Enhanced CLI & Configuration Tools ✅

**Released**: January 7, 2025

### Features
- ✅ **Enhanced Help & Completion**
  - Command completion (bash, zsh, fish, powershell)
  - Rich help examples for all commands
  - Man pages generation
- ✅ **Configuration Export Tools**
  - `niac config export` - Export and normalize configs
  - `niac config diff` - Compare configurations
  - `niac config merge` - Merge base and overlay
- ✅ **Documentation**
  - CLI reference guide (docs/CLI_REFERENCE.md)
  - CHANGELOG.md with full release history
  - Complete man pages (12 pages)

---

## Future Releases

### v2.0.0 - REST API & Basic Web
**Target**: 2-3 months

**Philosophy**: Start simple with Go templates and progressive enhancement, not a complex SPA.

#### REST API
- [ ] HTTP server with standard library (net/http or Chi/Echo)
- [ ] REST endpoints for device status, stats, config
- [ ] JSON responses for all data
- [ ] Basic authentication (optional)
- [ ] API versioning (/api/v1/)

#### Simple Web UI (Go Templates + HTMX)
- [ ] **Server-rendered HTML** with Go templates
- [ ] **HTMX** for dynamic updates (no build step!)
- [ ] Status dashboard (device list, stats)
- [ ] Real-time updates via SSE (Server-Sent Events) or HTMX polling
- [ ] Simple device controls (start/stop, error injection)
- [ ] Embedded in single binary (no separate frontend)

**Why HTMX over React?**
- No build step, no node_modules, no complexity
- Server-driven, fits Go's strengths
- Progressive enhancement from HTML
- Tiny (~14KB), works without JavaScript
- Perfect for admin/monitoring tools

### v2.1.0 - Enhanced Web Features
**Target**: 3-4 months

- [ ] WebSocket for true real-time updates (if SSE not sufficient)
- [ ] Device topology visualization (simple SVG/Canvas, no heavy library)
- [ ] Configuration editor (YAML textarea with validation)
- [ ] Log viewer with filtering
- [ ] Packet statistics graphs (simple charts, maybe Chart.js or pure SVG)
- [ ] Export stats as CSV/JSON

### v2.2.0 - Docker & Containers
**Target**: 4-6 months (lower priority)

- [ ] **Dockerfile** with multi-stage build
- [ ] Docker image published to Docker Hub
- [ ] Docker Compose example
- [ ] Health check endpoint
- [ ] Graceful shutdown handling
- [ ] Volume mounts for configs and walks
- [ ] Environment variable configuration
- [ ] Multi-architecture builds (amd64, arm64)

### v2.3.0+ - Kubernetes & Orchestration
**Target**: TBD (maybe never)

- [ ] Kubernetes manifests (if there's demand)
- [ ] Helm chart (if there's demand)
- [ ] Horizontal scaling support
- [ ] Service mesh compatibility

**Note**: Kubernetes support depends on actual user demand. Most users will run NIAC on a single host or simple Docker setup.

---

## Architecture Philosophy

**Embedded Everything**: Keep NIAC as a single binary with optional features.

```bash
# CLI mode (current)
niac interactive en0 config.yaml

# Web mode (v2.0.0+)
niac web en0 config.yaml
  └── Serves web UI on http://localhost:8080
  └── REST API on http://localhost:8080/api/v1/
  └── SSE updates on http://localhost:8080/events

# Docker mode (v2.1.0+)
docker run -p 8080:8080 niac-go web eth0 config.yaml
```

**Technology Stack (v2.0.0)**:
- **Backend**: Go net/http or Chi router
- **Templates**: Go html/template
- **Interactivity**: HTMX (~14KB) for AJAX-style updates
- **Styling**: Simple CSS (or Pico.css/Water.css for classless styling)
- **Real-time**: SSE (Server-Sent Events) - simpler than WebSocket
- **No build step**: Everything embedded in Go binary

---

## Version Breakdown Summary

| Version | Focus | Key Features | ETA |
|---------|-------|--------------|-----|
| **v1.0.0** | Core | Protocols, SNMP, Interactive, Simulation | ✅ DONE |
| **v1.5-v1.9** | Quality | Testing, Config, Code Quality | ✅ DONE |
| **v1.10.0** | Foundation | Cobra CLI, Validation, Error Reporting | ✅ DONE |
| **v1.11.0** | Templates | Pre-built configs, Quick start | ✅ DONE |
| **v1.12.0** | TUI | Interactive terminal UI, Monitoring | ✅ DONE |
| **v1.13.0** | CLI/Tools | Shell completion, Man pages, Config tools | ✅ DONE |
| **v2.0.0** | Web API | REST API, HTMX UI, Simple Dashboard | 2-3 months |
| **v2.1.0** | Web Enhanced | Charts, Topology, Editor | 3-4 months |
| **v2.2.0** | Containers | Docker images (lower priority) | 4-6 months |
| **v2.3.0+** | Kubernetes | K8s, Helm (if ever needed) | TBD |

---

## Future Ideas (v2.x+)

### Advanced Features
- [ ] Performance profiling and optimization
- [ ] Packet capture to PCAP file
- [ ] PCAP file replay
- [ ] Custom protocol definitions (plugin system)
- [ ] Network topology import (from real networks)
- [ ] Integration with network tools (Wireshark, tcpdump)
- [ ] Statistical analysis for traffic patterns
- [ ] Cloud deployment (AWS, GCP, Azure)
- [ ] SaaS offering (hosted NIAC)

### Protocol Enhancements
- [ ] BGP routing simulation
- [ ] OSPF routing simulation
- [ ] EIGRP routing simulation
- [ ] MPLS label switching
- [ ] VPN/IPSec simulation
- [ ] WiFi 802.11 simulation
- [ ] Bluetooth simulation
- [ ] LoRaWAN simulation
- [ ] 5G/LTE simulation

### Enterprise Features
- [ ] Multi-user support
- [ ] Role-based access control (RBAC)
- [ ] Audit logging
- [ ] SSO/LDAP integration
- [ ] High availability (HA) mode
- [ ] Distributed simulation (multiple hosts)
- [ ] Central management server
- [ ] Scenario library and sharing
- [ ] Compliance reporting

---

## Technology Stack (Future)

### Current (v1.0-1.2)
- **Language**: Go 1.21+
- **UI**: Terminal (Bubbletea)
- **Config**: Custom .cfg format
- **Deployment**: Native binary

### v1.3
- **Config**: JSON (primary), YAML (optional), .cfg (legacy)
- **Validation**: JSON Schema

### v2.0
- **Backend**: Go with Gin/Echo web framework
- **Frontend**: React/Vue/Svelte (TBD)
- **API**: REST + WebSocket
- **Container**: Docker, Kubernetes
- **Database**: SQLite (embedded) or PostgreSQL (optional)
- **Auth**: JWT tokens
- **Monitoring**: Prometheus metrics, Grafana dashboards

---

## Decision Points

### Config Format (v1.3)
**Status**: JSON recommended, YAML optional
- Primary: JSON for tooling, validation, web integration
- Secondary: YAML for human readability
- Legacy: CFG for backward compatibility

### Web UI Framework (v2.0)
**Status**: To be decided
**Options**:
1. **React** - Most popular, huge ecosystem
2. **Vue** - Easier learning curve, great docs
3. **Svelte** - Fastest, smallest bundles
4. **HTMX** - Minimal JS, server-driven (interesting for Go devs)

**Recommendation**: Start with HTMX for simplicity, migrate to React if needed

### Web UI Architecture (v2.0)
**Status**: Hybrid approach recommended
- Embed UI in binary for ease of use
- Support API-only mode for custom integrations
- Allow external UI connections

---

## Open Questions

1. **Config v1.3**: Should we support config encryption for sensitive data (SNMP communities, passwords)?
2. **Web UI v2.0**: Real-time vs polling for updates? (WebSocket vs HTTP polling)
3. **Containers v2.0**: Support rootless containers or require privileged mode?
4. **Cloud v2.x**: Should we build a hosted SaaS version?
5. **Licensing**: Keep open source or dual-license (open + commercial)?

---

## Community & Contribution

### v1.3 Goals
- [ ] Public GitHub repository
- [ ] CONTRIBUTING.md guide
- [ ] Issue templates
- [ ] Pull request templates
- [ ] Code of conduct
- [ ] Security policy
- [ ] Community Discord/Slack
- [ ] Documentation site

### v2.0 Goals
- [ ] Plugin system for custom protocols
- [ ] Marketplace for scenarios/configs
- [ ] Video tutorials
- [ ] Blog posts and articles
- [ ] Conference talks
- [ ] Academic partnerships (universities, research)

---

**Last Updated**: January 5, 2025
**Maintained By**: Kris Armstrong
**Status**: Living document, updated as project evolves
