# NIAC WebUI Design Mockup

## Overview

This document outlines a comprehensive UI redesign for NIAC (Network In A Can) WebUI to create a more polished,
professional, and user-friendly experience.

---

## Design System

### Color Palette

```text
Primary Brand Colors:
- Primary: #8B5CF6 (violet-500) - Main actions, active states
- Primary Hover: #7C3AED (violet-600)
- Primary Subtle: rgba(139, 92, 246, 0.1) - Backgrounds

Semantic Colors:
- Success: #10B981 (emerald-500)
- Warning: #F59E0B (amber-500)
- Error: #EF4444 (red-500)
- Info: #3B82F6 (blue-500)

Neutral Palette:
- Background: #030712 (gray-950)
- Surface: #111827 (gray-900)
- Surface Elevated: #1F2937 (gray-800)
- Border: rgba(255, 255, 255, 0.1)
- Border Focus: rgba(139, 92, 246, 0.5)

Text:
- Primary: #F9FAFB (gray-50)
- Secondary: #9CA3AF (gray-400)
- Muted: #6B7280 (gray-500)
- Inverse: #030712 (gray-950)
```

### Typography

```text
Font Families:
- Headings: 'Space Grotesk', sans-serif
- Body: 'Inter', system-ui, sans-serif
- Mono: 'JetBrains Mono', 'Fira Code', monospace

Scale:
- Display: 36px / 2.25rem, font-weight: 700
- H1: 28px / 1.75rem, font-weight: 700
- H2: 22px / 1.375rem, font-weight: 600
- H3: 18px / 1.125rem, font-weight: 600
- Body: 14px / 0.875rem, font-weight: 400
- Small: 12px / 0.75rem, font-weight: 400
- Micro: 10px / 0.625rem, font-weight: 500
```

### Spacing

```text
Base unit: 4px
- xs: 4px
- sm: 8px
- md: 16px
- lg: 24px
- xl: 32px
- 2xl: 48px
- 3xl: 64px
```

### Border Radius

```text
- sm: 6px (buttons, inputs)
- md: 8px (cards)
- lg: 12px (modals, large cards)
- xl: 16px (page sections)
- full: 9999px (pills, avatars)
```

---

## Layout Structure

### Current Issues

- Horizontal nav overflows on smaller screens
- No clear visual hierarchy between sections
- Inconsistent spacing and padding
- Navigation takes up too much horizontal space

### Proposed Layout

```text
┌──────────────────────────────────────────────────────────────────┐
│ ┌──────┐                                                         │
│ │ NIAC │  Command Center    Runtime    Devices    ...    [v2.12] │
│ └──────┘  ●                                                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Page Header                                    [Actions] │    │
│  │ Description text                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │ Stat Card        │  │ Stat Card        │  │ Stat Card      │ │
│  │                  │  │                  │  │                │ │
│  └──────────────────┘  └──────────────────┘  └────────────────┘ │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Main Content Area                                          │ │
│  │                                                            │ │
│  │                                                            │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Sidebar Navigation (Alternative for Complex Apps)

Consider a collapsible sidebar for better navigation:

```text
┌────────┬─────────────────────────────────────────────────────────┐
│ ≡ NIAC │ Page Title                                    [Actions] │
├────────┼─────────────────────────────────────────────────────────┤
│        │                                                         │
│ ● Dash │  Main Content                                          │
│ ◯ Ctrl │                                                         │
│ ◯ Dev  │                                                         │
│ ◯ Topo │                                                         │
│ ─────  │                                                         │
│ ◯ Anlys│                                                         │
│ ◯ Debug│                                                         │
│ ◯ Pkts │                                                         │
│        │                                                         │
│        │                                                         │
│ ─────  │                                                         │
│ v2.12  │                                                         │
└────────┴─────────────────────────────────────────────────────────┘
```

---

## Navigation Redesign

### Primary Navigation Improvements

1. **Grouped Navigation**: Organize items into logical groups
   - **Control**: Command Center, Runtime Control
   - **Configuration**: Devices & Config, Config Builder, Templates
   - **Network**: Topology & Neighbors, Traffic Injection
   - **Analysis**: Analysis, Debug Console, Packet Inspector, PCAP Analyzer
   - **Tools**: Config Diff, Automation

2. **Scrollable/Dropdown on overflow**: Add horizontal scroll or dropdown for nav items

3. **Active State Improvements**:

   ```css
   /* Current active */
   bg-violet-600/20 text-violet-300 border border-violet-500/30

   /* Proposed active - more prominent */
   bg-gradient-to-r from-violet-600/30 to-violet-500/20
   text-white
   border-l-2 border-violet-500
   shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]
   ```

4. **Add Breadcrumbs** for nested pages (e.g., Device Editor)

---

## Page-by-Page Improvements

### 1. Command Center (Dashboard)

**Current State**: Basic stat cards with counters

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Command Center                                                  │
│ Live counters and status for the active NIAC stack              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ STATUS OVERVIEW                                                 │
│ ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐    │
│ │ ● Running  │ │ 24 Devices │ │ 156 Pkts/s │ │ 99.9% Up   │    │
│ │ Simulation │ │ Active     │ │ Throughput │ │ Uptime     │    │
│ │ ▲ 12%      │ │ ▲ 2 new    │ │ ▼ 3%       │ │ → stable   │    │
│ └────────────┘ └────────────┘ └────────────┘ └────────────┘    │
│                                                                 │
│ ┌─────────────────────────────┐ ┌─────────────────────────────┐│
│ │ TRAFFIC GRAPH               │ │ QUICK ACTIONS               ││
│ │ ▁▂▃▅▆▇█▇▆▅▃▂▁▂▃▅▆▇        │ │ [▶ Start Simulation]        ││
│ │                             │ │ [⏸ Pause]  [⏹ Stop]         ││
│ │ Packets/sec over time       │ │ [↻ Reload Config]           ││
│ └─────────────────────────────┘ └─────────────────────────────┘│
│                                                                 │
│ RECENT ACTIVITY                                                 │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │ 10:32:15  Device router-1 sent LLDP frame                   ││
│ │ 10:32:14  DHCP lease granted to 192.168.1.100               ││
│ │ 10:32:12  SNMP walk completed for switch-core-1             ││
│ └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

**Key Changes**:

- Add trend indicators (▲▼→) to stat cards
- Mini sparkline graphs in stat cards
- Quick actions panel for common operations
- Real-time activity feed with timestamps
- Add a simple traffic graph visualization

### 2. Runtime Control

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Runtime Control                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ SIMULATION STATUS                                               │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │  ● RUNNING       Interface: en0       Config: enterprise.yml││
│ │  Started: 10:15:32 | Duration: 2h 15m 43s                   ││
│ │  ┌──────────────────────────────────────────────────┐       ││
│ │  │  [▶ Start] [⏸ Pause] [⏹ Stop] [↻ Restart]        │       ││
│ │  └──────────────────────────────────────────────────┘       ││
│ └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│ NETWORK INTERFACES                                              │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Interface    │ Status    │ Type     │ MAC Address          │ │
│ ├──────────────┼───────────┼──────────┼──────────────────────┤ │
│ │ ● en0        │ Active    │ Ethernet │ aa:bb:cc:dd:ee:ff    │ │
│ │ ○ lo0        │ Available │ Loopback │ 00:00:00:00:00:00    │ │
│ │ ○ utun0      │ Available │ Tunnel   │ —                    │ │
│ └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ LOADED CONFIGURATION                                            │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │ enterprise.yaml                           [View] [Reload]   ││
│ │ 24 devices | 6 routers | 12 switches | 6 servers           ││
│ └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 3. Devices & Config

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Devices & Configuration                                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ [Search...                    ] [Filter ▾] [+ Add Device]       │
│                                                                 │
│ VIEW: ● Cards  ○ Table  ○ Tree                                  │
│                                                                 │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │
│ │ 🔲 router-1  │ │ 🔲 switch-1  │ │ 🔲 server-1  │              │
│ │              │ │              │ │              │              │
│ │ Type: Router │ │ Type: Switch │ │ Type: Server │              │
│ │ IP: 10.0.0.1 │ │ IP: 10.0.0.2 │ │ IP: 10.0.0.3 │              │
│ │              │ │              │ │              │              │
│ │ SNMP LLDP    │ │ SNMP CDP STP │ │ SNMP HTTP    │              │
│ │              │ │              │ │              │              │
│ │ [Edit] [···] │ │ [Edit] [···] │ │ [Edit] [···] │              │
│ └──────────────┘ └──────────────┘ └──────────────┘              │
│                                                                 │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │
│ │ ...          │ │ ...          │ │ + Add Device │              │
│ └──────────────┘ └──────────────┘ └──────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

**Key Changes**:

- Multiple view modes (Cards, Table, Tree hierarchy)
- Device cards with protocol badges
- Quick action menu (···) for clone, delete, export
- Visual device type icons

### 4. Config Builder (Device Editor)

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ ← Back   Device: router-core-1                   [Save] [Reset] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ┌─────────────────┐                                             │
│ │ ● Basic Info    │ BASIC INFORMATION                           │
│ │ ○ Network       │ ┌─────────────────────────────────────────┐ │
│ │ ○ SNMP          │ │ Hostname: [router-core-1            ]   │ │
│ │ ○ Discovery     │ │ Type:     [Router              ▾]      │ │
│ │ ○ Services      │ │ MAC:      [aa:bb:cc:dd:ee:ff       ]   │ │
│ │ ○ Traffic       │ │ Primary IP: [192.168.1.1           ]   │ │
│ │ ○ Advanced      │ │                                         │ │
│ └─────────────────┘ │ Additional IPs:                         │ │
│                     │ [192.168.2.1] [×]                       │ │
│                     │ [10.0.0.1   ] [×]                       │ │
│                     │ [+ Add IP]                              │ │
│                     └─────────────────────────────────────────┘ │
│                                                                 │
│ YAML PREVIEW                               [Copy] [Download]    │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │ name: router-core-1                                         ││
│ │ type: router                                                ││
│ │ mac: aa:bb:cc:dd:ee:ff                                      ││
│ │ ip: 192.168.1.1                                             ││
│ │ ips:                                                        ││
│ │   - 192.168.2.1                                             ││
│ │   - 10.0.0.1                                                ││
│ └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

**Key Changes**:

- Vertical tabs for sections (much cleaner)
- Live YAML preview with syntax highlighting
- Inline validation messages
- Collapsible sections for protocols

### 5. Topology & Neighbors

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Topology & Neighbors                                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │                   INTERACTIVE TOPOLOGY                      │ │
│ │                                                            │ │
│ │         [router-1]────────[switch-1]                       │ │
│ │              │                  │                          │ │
│ │              │            ┌─────┴─────┐                    │ │
│ │         [router-2]   [server-1]  [server-2]                │ │
│ │                                                            │ │
│ │  [Zoom +] [Zoom -] [Fit] [Export SVG] [Export DOT]        │ │
│ └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ NEIGHBOR TABLE                                                  │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Local Device │ Protocol │ Remote Device │ Port  │ Details  │ │
│ ├──────────────┼──────────┼───────────────┼───────┼──────────┤ │
│ │ router-1     │ LLDP     │ switch-1      │ Gi0/1 │ [View]   │ │
│ │ router-1     │ CDP      │ switch-1      │ Gi0/1 │ [View]   │ │
│ │ switch-1     │ LLDP     │ server-1      │ Gi0/2 │ [View]   │ │
│ └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Key Changes**:

- Interactive topology diagram (consider vis.js or d3.js)
- Zoom and pan controls
- Export to multiple formats
- Clickable nodes for device details

### 6. Traffic Injection

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Traffic & Error Injection                                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ PCAP REPLAY                                                     │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │  ┌─────────────────────────────────────────────────────┐    ││
│ │  │     Drop PCAP file here or click to upload          │    ││
│ │  │              📁 Choose File                          │    ││
│ │  └─────────────────────────────────────────────────────┘    ││
│ │                                                             ││
│ │  Loaded: capture.pcap (1,234 packets)     [▶ Replay]        ││
│ │  ═══════════════════════════════●═══════════  45%           ││
│ │  Speed: [1x ▾]  Loop: [✓]  Interface: [en0 ▾]              ││
│ └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│ ERROR INJECTION                                                 │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │  ┌─────────────────────────────────────────────────────┐   ││
│ │  │ Error Type      │ Rate    │ Status   │ Action      │   ││
│ │  ├─────────────────┼─────────┼──────────┼─────────────┤   ││
│ │  │ Packet Loss     │ 5%      │ ● Active │ [Disable]   │   ││
│ │  │ Latency         │ 100ms   │ ○ Off    │ [Enable]    │   ││
│ │  │ Jitter          │ 20ms    │ ○ Off    │ [Enable]    │   ││
│ │  │ Corruption      │ 0.1%    │ ○ Off    │ [Enable]    │   ││
│ │  └─────────────────────────────────────────────────────┘   ││
│ └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 7. Debug Console

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Debug Console                                     [Clear] [⚙]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ PROTOCOL DEBUG LEVELS                                           │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │ SNMP   [━━━━━━━●━━━━]  Level: Debug                         ││
│ │ LLDP   [━━━●━━━━━━━━]  Level: Info                          ││
│ │ CDP    [━━━━━━━━━━●━]  Level: Trace                         ││
│ │ DHCP   [●━━━━━━━━━━━]  Level: Error                         ││
│ │ DNS    [━━━●━━━━━━━━]  Level: Info                          ││
│ └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│ LOG OUTPUT                          [Filter ▾] [Auto-scroll ✓]  │
│ ┌─────────────────────────────────────────────────────────────┐│
│ │ 10:32:15.123 [DEBUG] [SNMP] Processing GET request...       ││
│ │ 10:32:15.125 [INFO]  [LLDP] Sending frame from router-1     ││
│ │ 10:32:15.126 [DEBUG] [SNMP] OID: 1.3.6.1.2.1.1.1.0          ││
│ │ 10:32:15.130 [ERROR] [DHCP] Invalid option in request       ││
│ │ ▮                                                           ││
│ └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

**Key Changes**:

- Visual slider controls for debug levels
- Color-coded log levels
- Filter by protocol/level
- Auto-scroll toggle
- Log search functionality

### 8. Packet Inspector

**Proposed Improvements**:

```text
┌─────────────────────────────────────────────────────────────────┐
│ Packet Inspector                                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ┌───────────────────────────────────┬───────────────────────┐  │
│ │ PACKET LIST                       │ PACKET DETAILS        │  │
│ │                                   │                       │  │
│ │ # │ Time    │ Src      │ Protocol│ ▼ Frame               │  │
│ │───┼─────────┼──────────┼─────────│   Length: 64 bytes    │  │
│ │ 1 │ 0.000   │ 10.0.0.1 │ SNMP    │                       │  │
│ │ 2 │ 0.015   │ 10.0.0.2 │ LLDP    │ ▼ Ethernet II         │  │
│ │ 3 │ 0.023   │ 10.0.0.3 │ CDP     │   Src: aa:bb:cc:...   │  │
│ │ 4 │ 0.045   │ 10.0.0.1 │ DHCP    │   Dst: ff:ff:ff:...   │  │
│ │                                   │                       │  │
│ │                                   │ ▶ SNMP                │  │
│ │                                   │                       │  │
│ ├───────────────────────────────────┴───────────────────────┤  │
│ │ HEX DUMP                                                  │  │
│ │ 0000  ff ff ff ff ff ff aa bb  cc dd ee ff 08 00 45 00   │  │
│ │ 0010  00 34 12 34 40 00 40 11  b6 c0 0a 00 00 01 ff ff   │  │
│ └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Changes**:

- Wireshark-like 3-pane layout
- Collapsible protocol layers
- Hex dump with ASCII sidebar
- Packet highlighting by protocol

---

## Component Improvements

### Cards

```tsx
// Current: Basic dark cards
// Proposed: Glassmorphism with subtle gradients

<Card className="
  backdrop-blur-xl
  bg-gradient-to-br from-gray-900/90 to-gray-900/70
  border border-white/10
  shadow-xl shadow-black/20
  hover:shadow-violet-500/5
  transition-all duration-300
">
```

### Buttons

```tsx
// Primary button with gradient
<Button className="
  bg-gradient-to-r from-violet-600 to-violet-500
  hover:from-violet-500 hover:to-violet-400
  shadow-lg shadow-violet-500/25
  transition-all duration-200
">

// Ghost button with better hover
<Button variant="ghost" className="
  hover:bg-white/10
  active:bg-white/15
">
```

### Tables

```tsx
// Improved table styling
<table className="
  w-full
  [&_th]:bg-gray-950/60
  [&_th]:text-gray-400
  [&_th]:font-medium
  [&_th]:text-xs
  [&_th]:uppercase
  [&_th]:tracking-wider
  [&_td]:py-4
  [&_tr]:border-b
  [&_tr]:border-white/5
  [&_tr:hover]:bg-white/5
">
```

### Form Inputs

```tsx
// Improved input styling
<input className="
  w-full
  rounded-lg
  border border-white/10
  bg-gray-950/60
  px-4 py-2.5
  text-white
  placeholder:text-gray-500
  focus:border-violet-500
  focus:ring-2
  focus:ring-violet-500/20
  focus:outline-none
  transition-all
">
```

### Status Indicators

```tsx
// Animated pulse for active states
<span className="
  relative
  inline-flex
  h-3 w-3
  rounded-full
  bg-emerald-500
">
  <span className="
    absolute
    inline-flex
    h-full w-full
    animate-ping
    rounded-full
    bg-emerald-400
    opacity-75
  "></span>
</span>
```

---

## Animations & Transitions

### Page Transitions

```css
/* Fade in on page mount */
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.page-enter {
  animation: fadeIn 0.3s ease-out;
}
```

### Loading States

```css
/* Skeleton loading */
@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

.skeleton {
  background: linear-gradient(
    90deg,
    rgba(255,255,255,0.05) 25%,
    rgba(255,255,255,0.1) 50%,
    rgba(255,255,255,0.05) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
```

### Micro-interactions

- Button press: scale(0.98)
- Card hover: translateY(-2px), shadow increase
- Tab switch: underline slide animation
- Modal: scale + fade entrance

---

## Accessibility Improvements

1. **Focus States**: Visible focus rings on all interactive elements
2. **Color Contrast**: Ensure 4.5:1 minimum contrast ratio
3. **Keyboard Navigation**: Full keyboard support for all interactions
4. **ARIA Labels**: Proper labeling for screen readers
5. **Reduced Motion**: Respect `prefers-reduced-motion` media query

---

## Responsive Design

### Breakpoints

```text
- sm: 640px (mobile landscape)
- md: 768px (tablet)
- lg: 1024px (desktop)
- xl: 1280px (wide desktop)
- 2xl: 1536px (ultra-wide)
```

### Mobile Considerations

1. Collapsible sidebar navigation
2. Stacked cards instead of grids
3. Touch-friendly tap targets (min 44px)
4. Bottom navigation bar option
5. Swipe gestures for table rows

---

## Implementation Priority

### Phase 1: Foundation (High Impact)

1. ✅ Update color palette and spacing
2. ✅ Improve navigation (grouping, overflow handling)
3. ✅ Enhance Card components with glassmorphism
4. ✅ Add proper loading states (skeletons)
5. ✅ Improve form inputs and buttons

### Phase 2: Page Improvements

1. Command Center dashboard widgets
2. Device list card/table views
3. Config Builder vertical tabs
4. Debug console enhancements

### Phase 3: Advanced Features

1. Interactive topology visualization
2. Real-time traffic graphs
3. Advanced packet inspector
4. Animation system

---

## Technical Notes

### Dependencies to Consider

- `@tremor/react` - Dashboard charts and widgets
- `recharts` or `chart.js` - Traffic graphs
- `vis-network` or `cytoscape.js` - Topology visualization
- `monaco-editor` - YAML editing with syntax highlighting
- `framer-motion` - Animations

### Performance Considerations

- Virtual scrolling for long lists (already using)
- Lazy loading for heavy components
- Debounced search inputs
- Memoized expensive computations

---

## Example Component Updates

See individual component files for detailed implementation examples.
