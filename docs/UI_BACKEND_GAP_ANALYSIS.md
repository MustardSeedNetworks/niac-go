# UI/Backend Gap Analysis & Improvement Plan

## Overview

The UI has features that the backend doesn't support, and the page naming/structure needs improvement.

---

## 1. Missing Backend API Endpoints

### Templates API ✅ COMPLETE

Templates are the example YAML config files in `cmd/niac/templates/` and `examples/`.

| Endpoint | Method | Status |
|----------|--------|--------|
| `/api/v1/templates` | GET | ✅ Implemented |
| `/api/v1/templates/{name}` | GET | ✅ Implemented |
| `/api/v1/templates/use` | POST | ✅ Implemented |
| `/api/v1/templates` | POST | ⏳ Not Implemented |
| `/api/v1/templates/{name}` | DELETE | ⏳ Not Implemented |

**Implemented in:** `internal/api/templates.go`

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

### Proposed Renaming ✅ COMPLETE

| Path | New Name | Status |
|------|----------|--------|
| `/` | **Dashboard** | ✅ Done |
| `/runtime` | **Simulation** | ✅ Done |
| `/analysis` | **Replay** | ✅ Done |
| `/debug` | **Protocol Debug** | ✅ Done |
| `/packets` | **Live Packets** | ✅ Done |
| `/pcap-analyzer` | **PCAP Analysis** | ✅ Done |

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

### Merge Packet Inspector + PCAP Analyzer

**Current Problem:**
- Two separate pages for similar functionality
- Packet Inspector shows Wi-Fi symbol (hardcoded) even without Wi-Fi adapter
- No interface selection for live capture

**Proposed: Single "Traffic Analysis" Page**

```
┌─────────────────────────────────────────────────────────────┐
│ Traffic Analysis                                             │
├─────────────────────────────────────────────────────────────┤
│ Mode: ○ Live Capture  ○ Load PCAP                           │
│                                                              │
│ [Live Capture selected]                                      │
│ Interface: [dropdown: eth0, en0, etc.]  [▶ Start Capture]   │
│                                                              │
│ [Load PCAP selected]                                         │
│ File: [Browse...]  or drag & drop                            │
├─────────────────────────────────────────────────────────────┤
│ [Unified packet list / analysis view]                        │
└─────────────────────────────────────────────────────────────┘
```

**Backend Changes:**
- `/api/v1/interfaces` endpoint already exists - use it
- Return actual available interfaces, not hardcoded icons

**UI Changes:**
- Merge PacketInspectorPage + PcapAnalyzerPage
- Add interface dropdown populated from API
- Remove hardcoded Wi-Fi symbol
- Show actual interface type icon based on name (eth* = wired, wlan*/wlp* = wireless)

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

### Phase 1: Templates API ✅ COMPLETE
1. ✅ Add `/api/v1/templates` endpoint (list)
2. ✅ Add `/api/v1/templates/{name}` endpoint (get content)
3. ✅ Add `/api/v1/templates/use` endpoint (create from template)
4. ✅ Templates page works

### Phase 2: Server Configs (PENDING - awaiting discussion)
1. Add `/api/v1/configs` endpoint
2. Update RuntimeControlPage with dropdown
3. Connect dropdown to start simulation

### Phase 3: UI Cleanup ✅ PARTIAL
1. ✅ Rename pages (Dashboard, Simulation, Replay, etc.)
2. ✅ Fix error state persistence across pages
3. ⏳ Consider merge of Packet Inspector + PCAP Analyzer (awaiting discussion)
4. ⏳ Consider merge of Dashboard + Simulation

### Phase 4: Debug Levels API
1. Add debug levels endpoints
2. Debug Console page will work

---

## 5. Broken Features Summary

| Page | Issue | Fix |
|------|-------|-----|
| `/templates` | ✅ Fixed - Templates API implemented | Backend complete |
| `/topology` | Page Error (sometimes) | Needs config loaded first |
| `/analysis` | Replay unavailable | Expected - needs config |
| `/debug` | Debug levels don't work | Implement Debug Levels API |
| `/pcap-analyzer` | Open file does nothing | Needs investigation |

## 6. UI Bugs

### Error State Persistence Across Pages ✅ FIXED

**Issue:** When an API error occurs on one page, navigating to other pages may still show the error until the user manually reloads.

**Cause:** React state not being reset when navigating between pages.

**Fix Applied:**
- Added `PageWithErrorBoundary` wrapper in `App.tsx` that uses `location.pathname` as a key
- This forces React to unmount/remount the error boundary when navigating
- Each page now gets a fresh error boundary instance

---

## 7. Quick Wins

1. **Rename pages** - UI only, no backend changes
2. **Better error messages** - Show "No config loaded" instead of JSON errors
3. **Disable features when not applicable** - Gray out Topology when no config

---

*Created: 2026-01-26*
*Status: Planning*
