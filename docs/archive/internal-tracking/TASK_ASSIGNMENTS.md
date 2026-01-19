# NiAC Feature Parity - Developer Task Assignments

**Project:** NiAC 100% Interface Parity  
**Timeline:** 34 weeks  
**Start Date:** 2026-01-11  
**Status:** Ready for Assignment

---

## 📋 TASK BREAKDOWN BY DEVELOPER

### Developer 1: CLI & IPC Lead (17 days / 3.5 weeks)

**Priority:** P0 - Critical Path

#### Task 1.1: Complete IPC Socket Infrastructure ⚡ STARTED
**Status:** 50% Complete  
**Time:** 3 days remaining  
**Files:**
- ✅ `pkg/ipc/socket.go` (DONE - 10KB)
- ❌ `pkg/ipc/client.go` (TODO - see guide line 420)
- ❌ `pkg/ipc/socket_test.go` (TODO)
- ❌ `pkg/ipc/client_test.go` (TODO)

**Deliverables:**
- [x] Unix domain socket server with 6 commands
- [ ] IPC client with convenience methods
- [ ] Unit tests (80%+ coverage)
- [ ] Integration test (client→server)
- [ ] Documentation (`docs/IPC.md`)

**Blockers:** None  
**Blocks:** status, inject, monitor commands

---

#### Task 1.2: Implement `niac status` Command
**Status:** Not Started  
**Time:** 2 days  
**Dependencies:** IPC client complete  
**Files:**
- `cmd/niac/status.go` (see guide line 600)
- `cmd/niac/status_test.go`

**Deliverables:**
- [ ] Command with --json flag
- [ ] Human-readable output
- [ ] Exit codes (0/1/2)
- [ ] Custom socket path support
- [ ] Tests (5 test functions)

**Acceptance:**
- Run `niac status` and see "Status: RUNNING"
- Run `niac status --json` and get valid JSON
- Exit code 0 when running, 1 when stopped

---

#### Task 1.3: Implement `niac inject` Command
**Status:** Not Started  
**Time:** 3 days  
**Dependencies:** IPC client complete  
**Files:**
- `cmd/niac/inject.go` (see guide line 800)
- `cmd/niac/inject_test.go`

**Deliverables:**
- [ ] Inject subcommand (set error)
- [ ] List subcommand (show active)
- [ ] Clear subcommand (remove errors)
- [ ] JSON output for list
- [ ] Table output for list
- [ ] Tests (6 test functions)

**Acceptance:**
- Run `niac inject router1 high_cpu 85`
- Run `niac inject list` and see the injection
- Run `niac inject clear --all`

---

#### Task 1.4: Implement `niac monitor` Command
**Status:** Not Started  
**Time:** 4 days  
**Dependencies:** status command complete  
**Files:**
- `cmd/niac/monitor.go` (see guide line 1150)
- `pkg/monitor/streamer.go`
- `cmd/niac/monitor_test.go`

**Deliverables:**
- [ ] Real-time stat streaming
- [ ] Format: JSON, table, CSV
- [ ] Configurable interval
- [ ] Graceful Ctrl+C handling
- [ ] Rate calculations (pkt/sec)

**Acceptance:**
- Run `niac monitor en0 config.yaml` and see live stats
- Run `niac monitor en0 config.yaml --format json | jq`
- Ctrl+C exits cleanly

---

#### Task 1.5: Implement `niac logs tail` Command
**Status:** Not Started  
**Time:** 3 days  
**Dependencies:** IPC with log streaming support  
**Files:**
- `cmd/niac/logs_tail.go` (see guide line 1250)
- `pkg/logging/ring_buffer.go` (new)
- `cmd/niac/logs_tail_test.go`

**Deliverables:**
- [ ] Real-time log streaming
- [ ] Filter by level (0-3)
- [ ] Filter by protocol
- [ ] Color-coded output
- [ ] Tests

**Acceptance:**
- Run `niac logs tail en0 config.yaml --level 3`
- See colored log output in real-time
- Filter works: `--protocol snmp`

---

**Developer 1 Total:** 15 days (3 weeks)

---

### Developer 2: TUI Lead (25 days / 5 weeks)

**Priority:** P1 - High Value

#### Task 2.1: TUI Config Validation ([v] key)
**Status:** Not Started  
**Time:** 2 days  
**Files:**
- `pkg/interactive/config_validation.go` (see guide line 1350)
- Update `pkg/interactive/interactive.go` (keyboard handler)
- `pkg/interactive/config_validation_test.go`

**Deliverables:**
- [ ] Press [v] to validate config
- [ ] Modal window with results
- [ ] Color-coded (red/yellow/white)
- [ ] ESC to close
- [ ] Tests

**Acceptance:**
- Press [v] in TUI, see validation results
- Invalid config shows errors in red
- ESC closes the modal

---

#### Task 2.2: TUI Template Browser ([t] key)
**Status:** Not Started  
**Time:** 3 days  
**Files:**
- `pkg/interactive/template_browser.go` (see guide line 1450)
- Update `pkg/interactive/interactive.go`
- `pkg/interactive/template_browser_test.go`

**Deliverables:**
- [ ] Press [t] to open browser
- [ ] List all templates with descriptions
- [ ] Arrow keys to navigate
- [ ] Enter to preview
- [ ] [c] to copy path
- [ ] ESC to close

**Acceptance:**
- Press [t], see 7 templates listed
- Navigate with arrows
- Press Enter, see template preview
- Press [c], path copied to clipboard

---

#### Task 2.3: TUI Quick Edit ([e] key)
**Status:** Not Started  
**Time:** 4 days  
**Files:**
- `pkg/interactive/editor.go` (see guide line 1550)
- Update `pkg/interactive/interactive.go`
- `pkg/interactive/editor_test.go`

**Deliverables:**
- [ ] Press [e] to open $EDITOR
- [ ] Pause TUI during editing
- [ ] On save: validate config
- [ ] If valid: hot-reload
- [ ] If invalid: show errors, retry option

**Acceptance:**
- Press [e], vi opens with config
- Edit and save, TUI reloads config
- Invalid edit shows error, option to re-edit

---

#### Task 2.4: TUI PCAP Replay Control ([p] key)
**Status:** Not Started  
**Time:** 4 days  
**Files:**
- `pkg/interactive/pcap_replay.go`
- Update `pkg/interactive/interactive.go`
- `pkg/interactive/pcap_replay_test.go`

**Deliverables:**
- [ ] Press [p] to open PCAP panel
- [ ] List available PCAP files
- [ ] Start/stop/configure replay
- [ ] Show replay status in main view
- [ ] Tests

**Acceptance:**
- Press [p], see PCAP file list
- Select file, start replay
- See "Replaying: file.pcap" in status

---

#### Task 2.5: TUI Alert Configuration ([a] key)
**Status:** Not Started  
**Time:** 3 days  
**Files:**
- `pkg/interactive/alert_config.go`
- Update `pkg/interactive/interactive.go`
- `pkg/interactive/alert_config_test.go`

**Deliverables:**
- [ ] Press [a] to open alert panel
- [ ] Set packet threshold
- [ ] Configure webhook URL
- [ ] Show alert status in main view

**Acceptance:**
- Press [a], configure threshold
- Save, see "Alerts: threshold=10000" in status

---

#### Task 2.6: TUI Topology View ([T] key)
**Status:** Not Started  
**Time:** 5 days  
**Files:**
- `pkg/interactive/topology_view.go`
- `pkg/topology/ascii_renderer.go` (new package)
- Tests

**Deliverables:**
- [ ] Press [T] to show ASCII topology
- [ ] Device nodes with connections
- [ ] Trunk ports, port-channels
- [ ] Arrow keys to navigate
- [ ] ESC to close

**Acceptance:**
- Press [T], see ASCII diagram
- Shows device connections
- Navigate with arrows

---

#### Task 2.7: TUI Run History ([H] key)
**Status:** Not Started  
**Time:** 4 days  
**Files:**
- `pkg/interactive/history_viewer.go`
- Update `pkg/interactive/interactive.go`
- Tests

**Deliverables:**
- [ ] Press [H] to browse history
- [ ] Show last 20 runs from BoltDB
- [ ] Arrow keys to navigate
- [ ] Enter for details
- [ ] ESC to close

**Acceptance:**
- Press [H], see run history
- Navigate and view details

---

**Developer 2 Total:** 25 days (5 weeks)

---

### Developer 3: WebUI Lead (40 days / 8 weeks)

**Priority:** P0/P1 - Critical for WebUI parity

#### Task 3.1: ~~WebSocket~~ SSE Infrastructure ✅ COMPLETE
**Status:** COMPLETE
**Time:** 5 days
**Files:**
- `internal/api/sse.go` (implemented)
- `ui/src/hooks/useEventSource.ts` (implemented)
- Tests

**Deliverables:**
- [x] SSE hub (Go backend)
- [x] Endpoints: /api/v1/stream/logs, /api/v1/stream/stats
- [x] React hook for SSE connection
- [x] Auto-reconnect logic (browser native)
- [x] Rate limiting

**Acceptance:**
- Connect to /api/v1/stream/logs
- Receive log stream
- Auto-reconnect on disconnect (browser native)

---

#### Task 3.2: Packet Inspector Page
**Status:** Not Started
**Time:** 7 days
**Dependencies:** SSE infrastructure (DONE)  
**Files:**
- `webui/src/pages/PacketInspectorPage.tsx` (see guide line 1850)
- `webui/src/components/HexDumpViewer.tsx`
- `webui/src/components/PacketList.tsx`
- `webui/src/components/PacketDetails.tsx`

**Deliverables:**
- [ ] Real-time packet stream via SSE
- [ ] Packet list (last 100)
- [ ] Hex dump viewer
- [ ] Packet details (parsed fields)
- [ ] Protocol filtering
- [ ] Search functionality

**Acceptance:**
- Navigate to /packet-inspector
- See packets appearing in real-time
- Click packet, see hex dump
- Filter by protocol works

---

#### Task 3.3: Debug Console Page
**Status:** Not Started  
**Time:** 5 days  
**Dependencies:** SSE infrastructure (DONE)
**Files:**
- `webui/src/pages/DebugConsolePage.tsx` (see guide line 2000)
- `webui/src/components/LogViewer.tsx`
- `webui/src/components/LogFilters.tsx`

**Deliverables:**
- [ ] Real-time log stream via SSE
- [ ] Filter by level
- [ ] Filter by protocol
- [ ] Search/highlight
- [ ] Export to file
- [ ] Auto-scroll toggle

**Acceptance:**
- Navigate to /debug-console
- See logs streaming in real-time
- Filters work correctly
- Export downloads file

---

#### Task 3.4: Per-Protocol Debug Levels
**Status:** Not Started  
**Time:** 3 days  
**Files:**
- Update `webui/src/pages/RuntimeControlPage.tsx`
- `webui/src/components/ProtocolDebugPanel.tsx`
- Backend API endpoint (if needed)

**Deliverables:**
- [ ] UI for 19 protocol debug sliders
- [ ] Apply button sends to backend
- [ ] Visual feedback per protocol
- [ ] Save/load settings

**Acceptance:**
- Navigate to Runtime Control
- See 19 protocol debug sliders
- Adjust and apply, see logs change

---

#### Task 3.5: Config Diff/Merge Tool
**Status:** Not Started  
**Time:** 6 days  
**Files:**
- Add to `webui/src/pages/DevicesPage.tsx`
- `webui/src/components/ConfigDiffViewer.tsx`
- `webui/src/components/ConfigMergePanel.tsx`

**Deliverables:**
- [ ] Upload two configs to compare
- [ ] Side-by-side diff view
- [ ] Unified diff option
- [ ] Merge conflict resolution UI
- [ ] Export merged config

**Acceptance:**
- Upload two YAML files
- See diff highlighted
- Resolve conflicts, export result

---

#### Task 3.6: PCAP Analyzer
**Status:** Not Started  
**Time:** 6 days  
**Files:**
- `webui/src/pages/PcapAnalyzerPage.tsx`
- `webui/src/components/PcapUploader.tsx`
- `webui/src/components/ProtocolBreakdown.tsx`
- `webui/src/components/PacketTimeline.tsx`

**Deliverables:**
- [ ] Upload PCAP file
- [ ] Show protocol breakdown chart
- [ ] Timeline visualization
- [ ] Filter and search packets
- [ ] Export filtered results

**Acceptance:**
- Upload PCAP file
- See protocol pie chart
- Timeline shows packet timing
- Filter works

---

#### Task 3.7: SNMP Walk Analyzer
**Status:** Not Started  
**Time:** 5 days  
**Files:**
- Add to `webui/src/pages/AnalysisPage.tsx`
- `webui/src/components/SnmpWalkAnalyzer.tsx`
- `webui/src/components/OidTreeViewer.tsx`

**Deliverables:**
- [ ] Upload or select walk file
- [ ] Show device hierarchy
- [ ] Visualize OID tree
- [ ] Search functionality
- [ ] Export results

**Acceptance:**
- Upload walk file
- See OID tree
- Search for specific OIDs
- Export works

---

#### Task 3.8: Template Manager Page
**Status:** Not Started  
**Time:** 3 days  
**Files:**
- `webui/src/pages/TemplatesPage.tsx`
- `webui/src/components/TemplateCard.tsx`
- `webui/src/components/TemplatePreview.tsx`

**Deliverables:**
- [ ] List built-in templates (7)
- [ ] Preview template content
- [ ] Use template (creates new config)
- [ ] Upload custom templates
- [ ] Template search

**Acceptance:**
- Navigate to /templates
- See 7 built-in templates
- Click "Use", creates new config
- Upload works

---

**Developer 3 Total:** 40 days (8 weeks)

---

## 📊 TIMELINE VISUALIZATION

```
Week 1-4:   Dev1 [IPC + status + inject + monitor + logs]
Week 5-9:   Dev2 [TUI validation + templates + edit + PCAP + alerts]
Week 10-17: Dev3 [SSE + packets + logs + debug + diff + PCAP + SNMP + templates]
Week 18-20: ALL [Integration testing]
Week 21-24: ALL [Polish, docs, bug fixes]
Week 25-34: ALL [Advanced features (remote, plugins), final QA]
```

---

## 🎯 WEEKLY MILESTONES

### Week 1-2: Foundation
- ✅ IPC server/client complete
- ✅ `niac status` working
- ✅ `niac inject` working

### Week 3-4: CLI Complete
- ✅ `niac monitor` working
- ✅ `niac logs tail` working
- ✅ CLI at 100% parity

### Week 5-7: TUI Enhanced
- ✅ Config validation
- ✅ Template browser
- ✅ Quick edit

### Week 8-9: TUI Complete
- ✅ PCAP replay
- ✅ Alert config
- ✅ Topology view
- ✅ Run history

### Week 10-12: WebUI Foundation
- ✅ SSE infrastructure
- ✅ Packet Inspector page

### Week 13-15: WebUI Enhanced
- ✅ Debug Console page
- ✅ Per-protocol debug

### Week 16-17: WebUI Complete
- ✅ Config diff/merge
- ✅ PCAP analyzer
- ✅ SNMP analyzer
- ✅ Template manager
- ✅ WebUI at 100% parity

### Week 18-24: Polish & Testing
- ✅ Integration tests
- ✅ E2E tests
- ✅ Documentation
- ✅ Bug fixes

### Week 25-34: Advanced
- ✅ Remote operation
- ✅ Plugin system
- ✅ Performance optimization
- ✅ Final QA

---

## ✅ DAILY STANDUP FORMAT

**Each developer reports:**
1. What I completed yesterday
2. What I'm working on today
3. Any blockers

**Example:**
```
Dev1: Completed IPC client, starting status command. No blockers.
Dev2: Config validation 80% done, need review on modal UI. No blockers.
Dev3: SSE infrastructure working, starting packet inspector. No blockers.
```

---

## 📝 PULL REQUEST CHECKLIST

Before submitting PR, verify:
- [ ] Code compiles without errors
- [ ] All tests pass (`go test ./...`)
- [ ] Test coverage >80% for new code
- [ ] No linter warnings (`golangci-lint run`)
- [ ] Documentation updated
- [ ] CHANGELOG.md entry added
- [ ] Manual testing completed
- [ ] Screenshots/GIFs attached (for UI changes)

---

## 🆘 NEED HELP?

**Stuck on implementation?**
- Check `FEATURE_PARITY_IMPLEMENTATION_GUIDE.md` (27KB, comprehensive)
- Review existing code for patterns
- Ask in GitHub Discussions

**Test failures?**
- Run with `-v` flag for verbose output
- Check test fixtures and mocks
- Verify test isolation (no shared state)

**Integration issues?**
- Draw a diagram of the flow
- Add debug logging
- Test components in isolation first

---

## 🎉 CELEBRATION MILESTONES

- 🎊 Week 4: CLI at 100% parity!
- 🎊 Week 9: TUI at 100% parity!
- 🎊 Week 17: WebUI at 100% parity!
- 🎊 Week 24: All features complete!
- 🎊 Week 34: v2.0 RELEASE! 🚀

---

**Ready to start? Pick your role and begin with Task X.1!**

**Questions? See `FEATURE_PARITY_IMPLEMENTATION_GUIDE.md` for detailed specs.**
