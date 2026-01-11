# NiAC Feature Parity Implementation - Execution Plan

**Started:** 2026-01-11  
**Developer:** AI Assistant + Human Review  
**Estimated Completion:** 34 weeks

## Phase 1: Quick Wins (Weeks 1-12)

### Week 1-2: CLI Monitoring Commands ⚡ STARTING NOW

#### 1.1 `niac status` Command
**File:** `cmd/niac/status.go`
- Query running simulation via IPC socket
- Output: JSON or human-readable
- Exit codes: 0 (running), 1 (not running), 2 (error)

#### 1.2 `niac monitor` Command  
**File:** `cmd/niac/monitor.go`
- Stream statistics to stdout
- Formats: JSON, table, CSV
- Interval flag (default 1s)

#### 1.3 IPC Socket Infrastructure
**File:** `pkg/ipc/socket.go`
- Unix domain socket server
- Commands: status, reload, inject, clear
- Response format: JSON

### Week 3-4: Error Injection CLI

#### 2.1 `niac inject` Command
**File:** `cmd/niac/inject.go`
- Subcommands: set, list, clear
- Non-interactive error injection
- JSON output for list

### Week 5-7: TUI Configuration Tools

#### 3.1 Config Validation (v key)
**File:** `pkg/interactive/config_validator.go`
- Validate on demand
- Show errors in modal

#### 3.2 Template Browser (t key)
**File:** `pkg/interactive/template_browser.go`
- List templates with preview
- Copy to clipboard

#### 3.3 Quick Edit (e key)
**File:** `pkg/interactive/editor.go`
- Launch $EDITOR
- Validate and reload on save

### Week 8-12: WebUI Live Features

#### 4.1 WebSocket Infrastructure
**Files:** 
- `internal/server/websocket.go`
- `webui/src/hooks/useWebSocket.ts`

#### 4.2 Live Packet Inspector Page
**Files:**
- `webui/src/pages/PacketInspectorPage.tsx`
- `webui/src/components/HexDumpViewer.tsx`

#### 4.3 Live Debug Console Page
**Files:**
- `webui/src/pages/DebugConsolePage.tsx`
- `webui/src/components/LogViewer.tsx`

---

## Implementation Priority

### High Priority (Do First)
1. ✅ IPC socket infrastructure (enables CLI remote control)
2. ✅ `niac status` command (most requested)
3. ✅ `niac inject` command (critical for automation)
4. ✅ WebSocket infrastructure (enables WebUI live features)
5. ✅ TUI template browser (quick wins)

### Medium Priority (Do Next)
6. `niac monitor` command
7. TUI config validation
8. WebUI hex dump viewer
9. TUI PCAP replay control
10. WebUI debug console

### Lower Priority (Do Later)
11. `niac topology export`
12. TUI topology ASCII viewer
13. WebUI config diff tool
14. Remote operation mode
15. Plugin system

---

## Testing Strategy

Each feature gets:
- Unit tests (80%+ coverage)
- Integration test
- Manual QA checklist
- Documentation update

---

## Git Workflow

```bash
# Feature branches
git checkout -b feature/ipc-socket
git checkout -b feature/cli-status
git checkout -b feature/cli-inject
git checkout -b feature/tui-templates
git checkout -b feature/websocket-infra
git checkout -b feature/webui-hex-dump

# Merge to main after review
```

---

## Progress Tracking

- [ ] Phase 1.1: IPC Socket Infrastructure
- [ ] Phase 1.2: CLI Status Command
- [ ] Phase 1.3: CLI Inject Command
- [ ] Phase 1.4: CLI Monitor Command
- [ ] Phase 2.1: TUI Config Validation
- [ ] Phase 2.2: TUI Template Browser
- [ ] Phase 2.3: TUI Quick Edit
- [ ] Phase 3.1: WebSocket Infrastructure
- [ ] Phase 3.2: WebUI Packet Inspector
- [ ] Phase 3.3: WebUI Debug Console

