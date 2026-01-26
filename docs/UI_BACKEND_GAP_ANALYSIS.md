# UI/Backend Gap Analysis & Improvement Plan

## Overview

The UI has features that the backend doesn't support, and the page naming/structure needs improvement.

---

## 1. Missing Backend API Endpoints

### Templates API (HIGH PRIORITY)

Templates are the example YAML config files in `cmd/niac/templates/` and `examples/`.

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/templates` | GET | List available templates |
| `/api/v1/templates/{name}` | GET | Get template content |
| `/api/v1/templates/use` | POST | Create config from template |
| `/api/v1/templates` | POST | Upload new template |
| `/api/v1/templates/{name}` | DELETE | Delete template |

**Implementation Notes:**
- Scan `cmd/niac/templates/` and `examples/` directories
- Parse YAML frontmatter for metadata (name, description, type, tags)
- Return template list with metadata
- "Use" creates a copy in a configs directory

### Server Configs API (MEDIUM PRIORITY)

Allow selecting server-side configs instead of just local file browse.

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/configs` | GET | List configs on server |
| `/api/v1/configs/{name}` | GET | Get config content |

**Implementation Notes:**
- Scan configurable directory (e.g., `/etc/niac/configs/`, `~/.niac/configs/`)
- Return list for dropdown selection in Runtime Control

### Debug Levels API (LOW PRIORITY)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/debug/levels` | GET | Get protocol debug levels |
| `/api/v1/debug/levels` | PUT | Set debug levels |
| `/api/v1/debug/levels/reset` | POST | Reset to defaults |

---

## 2. UI Page Naming & Structure

### Current Structure (Confusing)

| Path | Current Name | What It Does |
|------|--------------|--------------|
| `/` | Command Center | Status dashboard (read-only monitoring) |
| `/runtime` | Runtime Control | Start/stop simulation, load configs |
| `/analysis` | Playback | PCAP replay and analysis |
| `/debug` | Debug Console | Protocol debug levels |
| `/packets` | Packet Inspector | Live packet viewing |
| `/pcap-analyzer` | PCAP Analyzer | Upload and analyze PCAP files |

### Proposed Renaming

| Path | New Name | Rationale |
|------|----------|-----------|
| `/` | **Dashboard** | Industry-standard name for status overview |
| `/runtime` | **Simulation** | Clear what it controls |
| `/analysis` | **Replay** | Direct description of function |
| `/debug` | **Protocol Debug** | More specific |
| `/packets` | **Live Packets** | Distinguishes from PCAP |
| `/pcap-analyzer` | **PCAP Analysis** | Consistent naming |

### Consider Merging

**Option A: Merge Dashboard + Simulation**
- Single page with status at top, controls below
- Path: `/`
- Name: "Control Center" or just "Dashboard"

**Option B: Keep Separate (Current)**
- Dashboard = monitoring
- Simulation = control
- Cleaner separation of concerns

**Recommendation:** Option A - Most users want status AND control in one place.

---

## 3. Feature: Server Config Selection

Currently, Runtime Control only has:
- Text input for config path
- File browse for local upload

**Add:**
- Dropdown showing configs available on server
- Populated from `/api/v1/configs` endpoint
- Can select from dropdown OR browse local OR enter path

```
┌─────────────────────────────────────────────────┐
│ Configuration                                    │
├─────────────────────────────────────────────────┤
│ ○ Server Config: [dropdown of available configs] │
│ ○ Local File:    [Browse...]                     │
│ ○ Path:          [text input]                    │
└─────────────────────────────────────────────────┘
```

---

## 4. Implementation Priority

### Phase 1: Templates API
1. Add `/api/v1/templates` endpoint (list)
2. Add `/api/v1/templates/{name}` endpoint (get content)
3. Add `/api/v1/templates/use` endpoint (create from template)
4. Templates page will work

### Phase 2: Server Configs
1. Add `/api/v1/configs` endpoint
2. Update RuntimeControlPage with dropdown
3. Connect dropdown to start simulation

### Phase 3: UI Cleanup
1. Rename pages
2. Consider merge of Dashboard + Runtime
3. Fix any remaining broken features

### Phase 4: Debug Levels API
1. Add debug levels endpoints
2. Debug Console page will work

---

## 5. Broken Features Summary

| Page | Issue | Fix |
|------|-------|-----|
| `/templates` | "Failed to Load Templates" | Implement Templates API |
| `/topology` | Page Error (sometimes) | Needs config loaded first |
| `/analysis` | Replay unavailable | Expected - needs config |
| `/debug` | Debug levels don't work | Implement Debug Levels API |
| `/pcap-analyzer` | Open file does nothing | Needs investigation |

---

## 6. Quick Wins

1. **Rename pages** - UI only, no backend changes
2. **Better error messages** - Show "No config loaded" instead of JSON errors
3. **Disable features when not applicable** - Gray out Topology when no config

---

*Created: 2026-01-26*
*Status: Planning*
