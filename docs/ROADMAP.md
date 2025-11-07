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

### v2.0.0 - Web UI & Containers
**Target**: 3-6 months

### Web UI Features
- [ ] **Modern web interface** (React/Vue/Svelte)
- [ ] Real-time dashboard
- [ ] Live packet visualization
- [ ] Interactive device map/topology
- [ ] Configuration editor (visual + code)
- [ ] Statistics graphs and charts
- [ ] Log viewer with filtering
- [ ] Error injection controls
- [ ] Device management (add/remove/edit)
- [ ] Traffic pattern controls
- [ ] WebSocket for real-time updates
- [ ] REST API for automation

### Container Features
- [ ] **Docker image** (`docker run niac-go`)
- [ ] Docker Compose for multi-device scenarios
- [ ] Kubernetes deployment manifests
- [ ] Helm charts
- [ ] Health checks and readiness probes
- [ ] Volume mounts for configs
- [ ] Environment-based configuration
- [ ] Multi-architecture support (amd64, arm64)

### Architecture: Web UI Options

#### **Option A: Embedded Web UI** (Recommended)
```
niac binary includes web server
├── Serve Web UI on http://localhost:8080
├── REST API on /api/*
├── WebSocket on /ws
└── Same binary, multiple modes:
    - CLI mode: niac --interactive en0 network.json
    - Web mode: niac --web en0 network.json
```

**Pros:**
- Single binary distribution
- No separate components
- Easy deployment
- Works offline

**Cons:**
- Larger binary size (~15-20 MB)
- UI updates require new binary

#### **Option B: Separate Web UI**
```
niac-server (Go backend with API)
└── REST API + WebSocket

niac-web (Frontend SPA)
└── React/Vue app served separately
```

**Pros:**
- Smaller core binary
- Independent UI updates
- Better separation of concerns

**Cons:**
- Multiple components to deploy
- More complex setup

#### **Option C: Hybrid** (Best of Both)
```
niac binary with embedded UI
└── Can also run as API-only server

External UI can connect if desired
└── For custom dashboards, integrations
```

### Container Deployment Example

```yaml
# docker-compose.yml
version: '3.8'
services:
  niac:
    image: krisarmstrong/niac-go:latest
    container_name: niac-simulator
    privileged: true  # Required for packet capture
    network_mode: host
    volumes:
      - ./configs:/configs
      - ./walks:/walks
    environment:
      - NIAC_INTERFACE=eth0
      - NIAC_CONFIG=/configs/network.json
      - NIAC_DEBUG_LEVEL=2
      - NIAC_WEB_ENABLED=true
      - NIAC_WEB_PORT=8080
    ports:
      - "8080:8080"  # Web UI
    command: --web --interface eth0 /configs/network.json
```

```bash
# Run with Docker
docker run -it --rm --privileged --net=host \
  -v $(pwd)/configs:/configs \
  -v $(pwd)/walks:/walks \
  krisarmstrong/niac-go:latest \
  --web --interface en0 /configs/network.json
```

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
| **v2.0.0** | Web/Cloud | Web UI, REST API, Docker/K8s | 3-6 months |

---

## Future Ideas (v2.x+)

### Advanced Features
- [ ] Performance profiling and optimization
- [ ] Packet capture to PCAP file
- [ ] PCAP file replay
- [ ] Custom protocol definitions (plugin system)
- [ ] Network topology import (from real networks)
- [ ] Integration with network tools (Wireshark, tcpdump)
- [ ] Machine learning for traffic patterns
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
