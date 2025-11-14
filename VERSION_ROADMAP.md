# NIAC-Go Version Roadmap

## v2.0.0 - Core WebUI with CLI Parity (READY FOR REVIEW)

### Status
**Ready for code review, QA, and documentation review before release**

### Completed Features
- ✅ Daemon mode architecture (#93)
- ✅ Full simulation lifecycle control from webUI (#94)
- ✅ Interface selection and management
- ✅ Configuration file upload and editing
- ✅ Real-time simulation status monitoring
- ✅ Network topology visualization (#30)
- ✅ Version information display
- ✅ Error injection type information
- ✅ Device and neighbor discovery display
- ✅ Runtime statistics and history

### Release Criteria
- [ ] Complete code review of all changes
- [ ] QA testing of daemon mode and webUI
- [ ] Documentation updates (README, API docs, user guide)
- [ ] Cross-platform testing (macOS, Linux, Windows)
- [ ] Performance validation

---

## v2.1.0 - Essential WebUI Feature Parity

### Goal
Complete the essential webUI features for full operational parity with CLI/TUI debugging and control capabilities.

### Issues (7 total)
1. **#86 - Live log streaming in webUI** (HIGH PRIORITY)
   - WebSocket/SSE for real-time logs
   - Log filtering and search
   - Log level controls
   - Essential for debugging running simulations

2. **#88 - Complete error injection API** (HIGH PRIORITY)
   - POST endpoint to inject errors by type
   - Error state management
   - Active error display in webUI
   - Currently only shows available types

3. **#87 - Hex dump packet viewer**
   - API endpoint for packet buffer
   - Hex viewer component
   - Packet navigation and details
   - Matches TUI 'x' key functionality

4. **#90 - Traffic injection controls**
   - Gratuitous ARP controls
   - ICMP ping generation
   - Random traffic (babble mode)
   - Per-device traffic patterns

5. **#76 - SNMP trap generation**
   - Generate traps on device state changes
   - Configurable trap destinations
   - Trap OID customization
   - Extends existing SNMP implementation

6. **#77 - VLAN-aware ARP responses**
   - ARP handling for tagged frames
   - VLAN configuration per device
   - 802.1Q support enhancement

7. **#89 - Monaco Editor for YAML editing**
   - Replace textarea with Monaco
   - Syntax highlighting
   - YAML validation
   - Better UX for config editing

### Estimated Timeline
4-6 weeks

---

## v2.2.0 - Advanced Analysis & Protocol Support

### Goal
Add advanced analysis tools and complete protocol implementations for production simulation scenarios.

### Issues (5 total)
1. **#91 - PCAP analysis and filtering tools**
   - Upload PCAP for analysis
   - Protocol breakdown visualization
   - Packet filtering and search
   - Export filtered packets
   - Matches `niac analyze-pcap` CLI command

2. **#92 - Configuration bundle export**
   - ZIP/TAR.GZ bundle creation
   - Include config, stats, PCAPs, walks
   - Metadata and manifest
   - Download from webUI

3. **#78 - Periodic traffic generation patterns**
   - Configurable traffic patterns
   - Time-based triggers
   - Traffic templates
   - Realistic network simulation

4. **#80 - Complete IPv6 protocol support**
   - IPv6 neighbor discovery
   - ICMPv6 implementation
   - DHCPv6 support
   - Dual-stack configuration

5. **#54 - Modern network equipment walk files**
   - Add SNMP walks for:
     - Arista switches
     - Cisco Nexus
     - Juniper QFX/EX
     - Modern wireless APs
   - Expand device simulation library

### Estimated Timeline
6-8 weeks

---

## v2.3.0 - Quality, Performance & Production Readiness

### Goal
Improve code quality, test coverage, and production deployment capabilities.

### Issues (6 total)
1. **#74 - HIGH priority test coverage**
   - pkg/capture (currently 22.8%)
   - pkg/device (currently 25.6%)
   - Target: 60%+ coverage

2. **#73 - MEDIUM priority test coverage**
   - cmd/niac-convert
   - internal/converter
   - Target: 50%+ coverage

3. **#55 - Overall test coverage to 40% threshold**
   - Currently below 40%
   - Add integration tests
   - Add end-to-end tests

4. **#47 - Low test coverage in core packages**
   - Identify gaps in core packages
   - Add unit tests
   - Mock external dependencies

5. **#36 - Performance monitoring and alerting**
   - Metrics collection (Prometheus format)
   - Performance dashboard
   - Alerting integration
   - Resource usage monitoring

6. **#35 - Container and Kubernetes deployment**
   - Dockerfile optimization
   - Kubernetes manifests
   - Helm chart
   - Multi-architecture builds (amd64, arm64)

### Estimated Timeline
8-10 weeks

---

## v2.4.0 - API Documentation & Hardening

### Goal
Formalize and document the existing REST API for third-party integrations and programmatic access.

### Issues (1 total)
1. **#31 - REST API documentation and hardening**
   - OpenAPI/Swagger documentation for all existing endpoints
   - API usage examples and tutorials
   - SDK generation (Python, Go, JavaScript)
   - Improve error responses and validation
   - API rate limiting (optional)
   - Better authentication options (currently just Bearer token)

   **Note:** We already have all the REST endpoints (#31 originally asked for). This is about documenting and hardening what exists, not building new functionality.

### Current API Endpoints (Already Implemented)
- ✅ `/api/v1/simulation` - Start/stop/status
- ✅ `/api/v1/devices` - Device listing
- ✅ `/api/v1/topology` - Network topology
- ✅ `/api/v1/stats` - Runtime statistics
- ✅ `/api/v1/config` - Configuration CRUD
- ✅ `/api/v1/neighbors` - Neighbor discovery
- ✅ `/api/v1/interfaces` - Interface listing
- ✅ `/api/v1/history` - Run history
- ✅ `/api/v1/replay` - PCAP replay control
- ✅ `/api/v1/alerts` - Alert configuration
- ✅ `/api/v1/files` - PCAP/walk file listing
- ✅ `/api/v1/version` - Version info
- ✅ `/api/v1/errors` - Error injection types
- ✅ `/api/v1/runtime` - Runtime status

### Estimated Timeline
3-4 weeks

---

## Future Releases (v3.0.0+)

### Major Features Requiring Architectural Changes (4 issues)

**Note:** #33 (Multi-user support) has been **deferred indefinitely** - not currently needed.

1. **#32 - Database persistence layer**
   - Replace/supplement BoltDB with PostgreSQL/MySQL option
   - Schema design for better querying
   - Migration strategy
   - Multi-instance support

2. **#34 - Advanced protocol analyzers**
   - Deep packet inspection
   - Protocol state machines
   - Anomaly detection
   - Custom protocol plugins

3. **#37 - Advanced network topology visualization**
   - Interactive graph with zoom/pan
   - Topology auto-layout algorithms
   - Link utilization visualization
   - Path tracing and highlighting
   - Real-time topology updates

4. **#33 - Multi-user support and authentication** (DEFERRED)
   - User management
   - Role-based access control (RBAC)
   - OAuth/OIDC integration
   - Session management
   - **Status:** Deferred indefinitely - single-user mode sufficient for current use cases

### Estimated Timeline
v3.0.0: TBD (2025 or later)

---

## Summary

| Version | Issues | Focus | Timeline |
|---------|--------|-------|----------|
| v2.0.0 | 0 open | Core CLI parity | **IN REVIEW** |
| v2.1.0 | 7 | Essential webUI features | 4-6 weeks |
| v2.2.0 | 5 | Advanced analysis & protocols | 6-8 weeks |
| v2.3.0 | 6 | Testing & production readiness | 8-10 weeks |
| v2.4.0 | 1 | API documentation & hardening | 3-4 weeks |
| v3.0.0+ | 3 active | Major architectural features | TBD 2025+ |

**Total Open Issues: 23**
**Issues covered in roadmap: 22** (1 deferred: #33)
**v2.x roadmap completes: 19 issues** ✅

---

## Release Process (for each version)

1. **Development Phase**
   - Create feature branches
   - Implement issues in priority order
   - Code review for each PR

2. **Testing Phase**
   - Unit tests (maintain/improve coverage)
   - Integration tests
   - Cross-platform testing
   - Performance testing

3. **Documentation Phase**
   - Update README
   - Update API documentation
   - Update user guide
   - Add migration notes if needed

4. **Release Phase**
   - Update CHANGELOG
   - Tag release
   - Build binaries for all platforms
   - Publish GitHub release
   - Update homebrew formula (if applicable)

---

## Version Naming Convention

- **Major (X.0.0)**: Breaking changes, major architectural updates
- **Minor (x.X.0)**: New features, backward compatible
- **Patch (x.x.X)**: Bug fixes, minor improvements

Current series: **v2.x.x** (WebUI generation)
Next major: **v3.x.x** (Multi-user/enterprise features)
