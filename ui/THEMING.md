# NIAC Theme System

> Design system documentation for the NIAC Network Infrastructure Analyzer UI.

## Overview

NIAC uses CSS custom properties (variables) for theming, defined in `src/index.css`. The design follows a dark-first approach with high contrast for accessibility (WCAG AA compliant).

## Color Palette

### Brand Colors

Violet-based brand identity with high-contrast variants:

| Token | Value | Usage |
|-------|-------|-------|
| `--color-brand-50` | `#f5f3ff` | Lightest background accent |
| `--color-brand-100` | `#ede9fe` | Light hover states |
| `--color-brand-200` | `#ddd6fe` | Light borders |
| `--color-brand-300` | `#c4b5fd` | Secondary accents |
| `--color-brand-400` | `#a78bfa` | Primary hover |
| `--color-brand-500` | `#8b5cf6` | **Primary brand color** |
| `--color-brand-600` | `#7c3aed` | Primary pressed |
| `--color-brand-700` | `#6d28d9` | Deep accent |
| `--color-brand-800` | `#5b21b6` | Darker accent |
| `--color-brand-900` | `#4c1d95` | Darkest accent |

### Surface Colors

Dark theme backgrounds with visual hierarchy:

| Token | Value | Usage |
|-------|-------|-------|
| `--color-bg-base` | `#030712` | Deepest background (body) |
| `--color-bg-surface` | `#0f1117` | Main content surface |
| `--color-bg-elevated` | `#1a1d25` | Cards, modals |
| `--color-bg-overlay` | `#242731` | Dropdowns, tooltips |
| `--color-bg-muted` | `#2d3139` | Inactive/disabled states |

### Text Colors

High contrast text for accessibility:

| Token | Value | Usage |
|-------|-------|-------|
| `--color-text-primary` | `#f8fafc` | Primary text (high contrast) |
| `--color-text-secondary` | `#cbd5e1` | Secondary text |
| `--color-text-muted` | `#94a3b8` | Muted/placeholder |
| `--color-text-disabled` | `#64748b` | Disabled states |
| `--color-text-inverse` | `#0f172a` | Text on light backgrounds |

### Border Colors

Subtle borders with focus states:

| Token | Value | Usage |
|-------|-------|-------|
| `--color-border-default` | `rgba(255, 255, 255, 0.08)` | Default borders |
| `--color-border-subtle` | `rgba(255, 255, 255, 0.05)` | Subtle dividers |
| `--color-border-muted` | `rgba(255, 255, 255, 0.12)` | Hover borders |
| `--color-border-focus` | `rgba(139, 92, 246, 0.6)` | Focus ring |
| `--color-border-error` | `rgba(239, 68, 68, 0.5)` | Error states |

### Status Colors

Semantic colors for feedback:

#### Success (Green)
- `--color-success-400`: `#4ade80` - Primary success text
- `--color-success-500`: `#22c55e` - Success background
- `--color-success-600`: `#16a34a` - Success pressed

#### Warning (Amber)
- `--color-warning-400`: `#fbbf24` - Primary warning text
- `--color-warning-500`: `#f59e0b` - Warning background
- `--color-warning-600`: `#d97706` - Warning pressed

#### Error (Red)
- `--color-error-400`: `#f87171` - Primary error text
- `--color-error-500`: `#ef4444` - Error background
- `--color-error-600`: `#dc2626` - Error pressed

#### Info (Blue)
- `--color-info-400`: `#60a5fa` - Primary info text
- `--color-info-500`: `#3b82f6` - Info background
- `--color-info-600`: `#2563eb` - Info pressed

## Network-Specific Colors

### Device Type Colors (Topology)

Distinct colors for network device identification:

| Token | Color | Device Type |
|-------|-------|-------------|
| `--color-device-router` | `#3b82f6` (Blue) | Routers |
| `--color-device-switch` | `#22c55e` (Green) | Switches |
| `--color-device-firewall` | `#ef4444` (Red) | Firewalls |
| `--color-device-server` | `#f97316` (Orange) | Servers |
| `--color-device-workstation` | `#6b7280` (Gray) | Workstations |
| `--color-device-ap` | `#a855f7` (Purple) | Access Points |
| `--color-device-iot` | `#14b8a6` (Teal) | IoT Devices |
| `--color-device-unknown` | `#94a3b8` (Slate) | Unknown |

### Link Speed Colors

Color-coded network speeds:

| Token | Color | Speed |
|-------|-------|-------|
| `--color-link-10m` | `#94a3b8` (Gray) | 10 Mbps |
| `--color-link-100m` | `#22c55e` (Green) | 100 Mbps |
| `--color-link-1g` | `#3b82f6` (Blue) | 1 Gbps |
| `--color-link-10g` | `#a855f7` (Purple) | 10 Gbps |
| `--color-link-25g` | `#f97316` (Orange) | 25 Gbps |
| `--color-link-40g` | `#ec4899` (Pink) | 40 Gbps |
| `--color-link-100g` | `#eab308` (Yellow) | 100 Gbps |
| `--color-link-trunk` | `#06b6d4` (Cyan) | Trunk/LAG |

### Protocol Colors

Protocol identification in logs/debug:

| Token | Color | Protocol |
|-------|-------|----------|
| `--color-proto-arp` | `#22c55e` | ARP |
| `--color-proto-icmp` | `#3b82f6` | ICMP |
| `--color-proto-dns` | `#a855f7` | DNS |
| `--color-proto-dhcp` | `#f97316` | DHCP |
| `--color-proto-snmp` | `#14b8a6` | SNMP |
| `--color-proto-lldp` | `#ec4899` | LLDP |
| `--color-proto-cdp` | `#eab308` | CDP |
| `--color-proto-http` | `#6366f1` | HTTP |
| `--color-proto-tcp` | `#06b6d4` | TCP |
| `--color-proto-udp` | `#8b5cf6` | UDP |

## Typography

### Font Families

```css
/* Display headings */
font-family: 'Space Grotesk', sans-serif;

/* Body text */
font-family: 'Inter', system-ui, sans-serif;

/* Code/monospace */
font-family: 'JetBrains Mono', 'Fira Code', monospace;
```

## Component Classes

### Buttons

```css
.btn-primary   /* Gradient violet, primary actions */
.btn-secondary /* Transparent white, secondary actions */
.btn-ghost     /* No background, tertiary actions */
.btn-outline   /* Border only, subtle actions */
```

### Cards

```css
.glass-card       /* Elevated surface with blur */
.glass-card-hover /* Glass card with hover effects */
.card-surface     /* Flat elevated background */
.card-elevated    /* Higher elevation with shadow */
```

### Status Badges

```css
.badge-success /* Green status indicator */
.badge-warning /* Amber status indicator */
.badge-error   /* Red status indicator */
.badge-info    /* Blue status indicator */
```

### Inputs

```css
.input-base    /* Standard input styling */
.focus-ring    /* Focus state ring */
```

## Usage Examples

### Using CSS Variables in Components

```tsx
// In TSX with inline styles
<div style={{ color: 'var(--color-text-primary)' }}>
  Primary text
</div>

// In Tailwind (using arbitrary values)
<div className="text-[var(--color-text-primary)]">
  Primary text
</div>

// Recommended: Use Tailwind utilities that map to these vars
<div className="text-white bg-[var(--color-bg-elevated)]">
  Content
</div>
```

### Device Type Badge

```tsx
const deviceTypeColors: Record<DeviceType, string> = {
  router: 'var(--color-device-router)',
  switch: 'var(--color-device-switch)',
  firewall: 'var(--color-device-firewall)',
  server: 'var(--color-device-server)',
  workstation: 'var(--color-device-workstation)',
  access_point: 'var(--color-device-ap)',
  iot: 'var(--color-device-iot)',
  unknown: 'var(--color-device-unknown)',
};
```

## Accessibility

- All text colors meet WCAG AA contrast requirements (4.5:1 for normal text)
- Focus states use visible ring with brand color
- Reduced motion preference is respected
- Status colors include non-color indicators (icons, patterns)

## Related Files

- `src/index.css` - CSS variable definitions
- `src/components/ui/` - UI component implementations
- `DESIGN_MOCKUP.md` - Visual design reference
