# NIAC-Go Roadmap

## Current Release: v1.0.0 ✅
- Complete protocol stack
- Interactive error injection
- Device simulation
- Traffic generation
- **Status**: SHIPPED

---

## v1.1.0 - Enhanced CLI (In Progress)
**Target**: 1-2 weeks

### Features
- [ ] Enhanced CLI flags (--version, --list-interfaces, --dry-run, --verbose, --quiet, etc.)
- [ ] Interactive mode improvements (debug level cycling, log viewer, stats viewer, help)
- [ ] Debug level in status bar
- [ ] Color-coded output
- [ ] Per-protocol debug levels

**ETA**: 6-8 hours of work remaining

---

## v1.2.0 - Full Protocol Parity
**Target**: 2-3 weeks

### Features
- [ ] IPv6 and ICMPv6 support
- [ ] NetBIOS protocol support
- [ ] Spanning Tree Protocol (STP)
- [ ] Complete Java NIAC feature parity

**Result**: Go NIAC becomes definitively superior to Java in ALL categories

---

## v1.3.0 - Enhanced Configuration
**Target**: 1-2 months

### Configuration Enhancements
- [ ] **CFG to JSON converter** (automatic migration tool)
- [ ] **JSON config file support** (primary format)
- [ ] **YAML config file support** (alternative format)
- [ ] Backward compatibility with .cfg files
- [ ] Config validation and linting
- [ ] Config templates for common scenarios
- [ ] Config generator with wizard
- [ ] Environment variable substitution
- [ ] Include/import directives
- [ ] Config inheritance and overlays

### Why JSON Over YAML?

#### ✅ **JSON Advantages**
1. **Native Go support** - `encoding/json` in stdlib (no dependencies)
2. **Validation** - Strong typing, schema validation (JSON Schema)
3. **Tooling** - Better IDE support, linters, formatters
4. **Parsing speed** - Faster than YAML parsing
5. **Strict syntax** - Less ambiguous than YAML
6. **Web compatibility** - Direct use in Web UI (future v2.0)
7. **API friendly** - REST APIs, configuration APIs
8. **Size** - Slightly smaller than YAML
9. **Security** - No code execution risks (YAML has had security issues)

#### ⚠️ **YAML Advantages**
1. **Human-readable** - Cleaner for large configs (no braces/quotes)
2. **Comments** - Native comment support (JSON needs workarounds)
3. **Multi-line strings** - Easier to read
4. **Less verbose** - No closing braces
5. **Anchors & aliases** - Config reuse (though can be confusing)

#### 🎯 **Recommendation: JSON as Primary, YAML as Optional**

**Strategy:**
```
Primary:   JSON (best for tooling, validation, web integration)
Optional:  YAML (for users who prefer readability)
Legacy:    CFG  (backward compatibility, no new features)
```

**Example JSON Config:**
```json
{
  "version": "2.0",
  "network": {
    "name": "test-network",
    "subnet": "192.168.1.0/24"
  },
  "devices": [
    {
      "name": "Router1",
      "type": "router",
      "ip": "192.168.1.1",
      "mac": "00:11:22:33:44:55",
      "snmp": {
        "community": "public",
        "sysname": "Router1",
        "sysdescr": "Cisco IOS Router",
        "walk_file": "walks/cisco_router.walk"
      },
      "interfaces": [
        {
          "name": "eth0",
          "speed": "1000M",
          "duplex": "full",
          "admin_status": "up"
        }
      ]
    }
  ]
}
```

**Conversion Tool:**
```bash
# Convert old .cfg to new .json
niac config convert network.cfg network.json

# Validate config
niac config validate network.json

# Generate from template
niac config generate --template router --count 5 > routers.json
```

---

## v2.0.0 - Web UI & Containers
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
| **v1.1.0** | CLI | Enhanced flags, debug tools, help | 1-2 weeks |
| **v1.2.0** | Parity | IPv6, NetBIOS, STP | 2-3 weeks |
| **v1.3.0** | Config | JSON/YAML support, converter, validation | 1-2 months |
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
