# NIAC Theme System

> Design system documentation for the NIAC (Network In A Can) UI by **Mustard Seed Networks**.

> [!IMPORTANT]
> **Describes canonical TARGET state — locked 2026-05-22 (brand audit).**
> Pre-audit, this document described a violet brand identity that no longer applies. NIAC's anchor
> is now Tailwind **indigo-600** (`#4f46e5`) — engineering / infrastructure tone, distinct from
> Seed's green and Stem's blue. The actual `src/index.css` may still hold older values; the brand
> migration is phased (1–7) and lands in order: status palette → fonts → typography → surfaces →
> per-product brand → modules → cleanup.
>
> - **Source-of-truth Tailwind v4 `@theme` block:** `_web/mustardseednetworks-com/src/index.css`
> - **Cross-product canonical map:** `msn-docs-internal/04-Brand-Marketing/CANONICAL_TOKENS.md`

## Architecture Overview

The theming system has three layers:

1. **CSS Variables** (`src/index.css`) — Core color tokens for light/dark modes.
2. **TypeScript tokens** (`src/styles/theme.ts` and friends) — Re-exports of role tokens for use in TSX.
3. **Tailwind v4 `@theme` block** — maps CSS variables to utility classes (`bg-brand-primary`, etc).

## Color Palette

### Brand Colors

Brand anchors are **constant across light and dark modes** (do not lighten in dark). Foreground
variants (`-strong` for text on light surfaces, `-soft` for text on dark) shift to preserve AA.

| Token                          | Value     | Usage                                                                |
| ------------------------------ | --------- | -------------------------------------------------------------------- |
| `--color-brand-primary`        | `#4f46e5` | NIAC Indigo anchor (indigo-600) — filled buttons, focus rings, glows |
| `--color-brand-primary-strong` | `#3730a3` | Darker (indigo-700) — text/links on light surfaces (AA)              |
| `--color-brand-primary-soft`   | `#a5b4fc` | Lighter (indigo-300) — text/links on dark surfaces (AA)              |
| `--color-brand-gold`           | `#d4a017` | Mustard cross-brand accent — warning state, focus, premium highlights |

#### Indigo ramp (for component composition)

| Token             | Value     |
| ----------------- | --------- |
| `--color-niac-50`  | `#eef2ff` |
| `--color-niac-100` | `#e0e7ff` |
| `--color-niac-300` | `#a5b4fc` |
| `--color-niac-400` | `#818cf8` |
| `--color-niac-500` | `#6366f1` |
| `--color-niac-600` | `#4f46e5` (= `--color-brand-primary`) |
| `--color-niac-700` | `#3730a3` |
| `--color-niac-900` | `#1e1b4b` |

### Module Accents (Icons / Badges / Legends)

Five differentiated hues for NIAC's feature modules — replaces the pre-audit mono-indigo ramp that
made the modules visually indistinguishable. Mirrors how Seed (Roots/Canopy/Shell/Sap/Harvest) and
Stem (Reflector/Benchmark/ServiceTest/TrafficGen/Measure/Certify) differentiate.

| Token                       | Value     | Module      | Domain                                |
| --------------------------- | --------- | ----------- | ------------------------------------- |
| `--color-module-topology`   | `#4f46e5` | Topology    | Network map / graph view (= brand anchor) |
| `--color-module-protocols`  | `#0d9488` | Protocols   | Protocol stack, packet types          |
| `--color-module-analyze`    | `#c026d3` | Analyze     | Capture inspection, synthesis         |
| `--color-module-inject`     | `#e11d48` | Inject      | Traffic generation, replay            |
| `--color-module-templates`  | `#d97706` | Templates   | Saved configs, library                |

Module accents are **constant across light and dark**. Used only for icons, badge fills, and legend
swatches — never for card backgrounds.

### Surface Colors — botanical-earth palette

Warm cream in light, deep green-black in dark. Replaces the prior cool-slate / neutral-charcoal
palette across all three MSN products for visual coherence.

| Token                    | Light     | Dark      | Usage                       |
| ------------------------ | --------- | --------- | --------------------------- |
| `--color-surface`        | `#fbfaf5` | `#0e1612` | Page background             |
| `--color-surface-raised` | `#f3efe2` | `#16201b` | Subtle elevation, hover bg  |
| `--color-surface-sunken` | `#e9e3ce` | `#1f2c25` | Inset / recessed areas      |
| `--color-card`           | `#ffffff` | `#1a2520` | Cards, modals               |
| `--color-border`         | `#e1d9c4` | `#2b3a31` | Default border              |
| `--color-border-strong`  | `#b6ab8b` | `#3d4f44` | Emphasized border           |

### Text Colors

WCAG AA passing on the matching surface.

| Token                  | Light     | Dark      | Usage                              |
| ---------------------- | --------- | --------- | ---------------------------------- |
| `--color-fg`           | `#1a2520` | `#e8eee9` | Primary text                       |
| `--color-fg-muted`     | `#4a5650` | `#a3b1a7` | Secondary text                     |
| `--color-fg-subtle`    | `#6b766f` | `#9aa297` | Tertiary text, metadata            |
| `--color-text-inverse` | `#ffffff` | `#1a2520` | Text on filled brand backgrounds   |

### Status Colors

Status anchors tied to the brand: `success = seed-500`, `warning = mustard-500`, `info = stem-500`.
Constant across modes; foreground/bg variants shift for AA.

| Token              | Value     | Usage                                              |
| ------------------ | --------- | -------------------------------------------------- |
| `--color-success`  | `#4caf50` | Positive states (= seed-500, shared cross-product) |
| `--color-warning`  | `#d4a017` | Caution states (= mustard-500 brand cross-accent)  |
| `--color-danger`   | `#ef4444` | Error / destructive actions                        |
| `--color-info`     | `#1976d2` | Informational states (= stem-500)                  |

## Network-Specific Colors

These are **orthogonal to the brand** — they identify domain concepts (a router is blue because it's
a router, not because of brand). They live in `themeDeviceColors.ts`.

### Device Type Colors (Topology view)

| Token                        | Color              | Device Type   |
| ---------------------------- | ------------------ | ------------- |
| `--color-device-router`      | `#3b82f6` Blue     | Routers       |
| `--color-device-switch`      | `#22c55e` Green    | Switches      |
| `--color-device-firewall`    | `#ef4444` Red      | Firewalls     |
| `--color-device-server`      | `#f97316` Orange   | Servers       |
| `--color-device-workstation` | `#6b7280` Gray     | Workstations  |
| `--color-device-ap`          | `#a855f7` Purple   | Access Points |
| `--color-device-iot`         | `#14b8a6` Teal     | IoT devices   |
| `--color-device-unknown`     | `#94a3b8` Slate    | Unknown       |

### Link Speed Colors

| Token                | Color    | Speed     |
| -------------------- | -------- | --------- |
| `--color-link-10m`   | `#94a3b8` | 10 Mbps  |
| `--color-link-100m`  | `#22c55e` | 100 Mbps |
| `--color-link-1g`    | `#3b82f6` | 1 Gbps   |
| `--color-link-10g`   | `#a855f7` | 10 Gbps  |
| `--color-link-25g`   | `#f97316` | 25 Gbps  |
| `--color-link-40g`   | `#ec4899` | 40 Gbps  |
| `--color-link-100g`  | `#eab308` | 100 Gbps |
| `--color-link-trunk` | `#06b6d4` | Trunk / LAG |

### Protocol Colors

| Token                | Color    | Protocol |
| -------------------- | -------- | -------- |
| `--color-proto-arp`  | `#22c55e` | ARP     |
| `--color-proto-icmp` | `#3b82f6` | ICMP    |
| `--color-proto-dns`  | `#a855f7` | DNS     |
| `--color-proto-dhcp` | `#f97316` | DHCP    |
| `--color-proto-snmp` | `#14b8a6` | SNMP    |
| `--color-proto-lldp` | `#ec4899` | LLDP    |
| `--color-proto-cdp`  | `#eab308` | CDP     |
| `--color-proto-http` | `#6366f1` | HTTP    |
| `--color-proto-tcp`  | `#06b6d4` | TCP     |
| `--color-proto-udp`  | `#8b5cf6` | UDP     |

## Typography

Self-hosted via `@fontsource-variable/inter` and `@fontsource-variable/jetbrains-mono` (no Google
Fonts dependency, no FOUT, CSP-safe). **Space Grotesk is no longer used** anywhere — Inter covers
both display and body.

```css
--font-sans: "Inter Variable", "Inter", system-ui, sans-serif;
--font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, monospace;
```

### Type scale

| Class           | Mobile      | Desktop          | Weight   | Leading        | Tracking         |
| --------------- | ----------- | ---------------- | -------- | -------------- | ---------------- |
| `.heading-1`    | text-2xl    | sm:text-3xl      | bold     | leading-tight  | tracking-tight   |
| `.heading-2`    | text-xl     | sm:text-2xl      | semibold | leading-snug   | —                |
| `.heading-3`    | text-lg     | sm:text-xl       | semibold | leading-snug   | —                |
| `.heading-4`    | text-base   | sm:text-lg       | medium   | leading-snug   | —                |
| `.section-title`| text-xs UC  | —                | medium   | leading-normal | tracking-wider   |
| `.body-large`   | text-lg     | —                | normal   | leading-relaxed| —                |
| `.body`         | text-base   | —                | normal   | leading-relaxed| —                |
| `.body-small`   | text-sm     | —                | normal   | leading-relaxed| —                |
| `.caption`      | text-xs     | —                | normal   | leading-normal | —                |
| `.label`        | text-sm     | —                | medium   | —              | —                |
| `.code`         | text-sm     | —                | normal   | —              | (font-mono)      |

## Component Classes

| Class            | Purpose                                                |
| ---------------- | ------------------------------------------------------ |
| `.btn-primary`   | Indigo-filled CTA — uses `text-text-inverse` (white on indigo) |
| `.btn-secondary` | Surface-raised with border                             |
| `.btn-ghost`     | Transparent until hover                                |
| `.btn-outline`   | Border only                                            |
| `.card-surface`  | Flat default card (canonical — matches Seed/Stem)      |
| `.card-elevated` | With shadow                                            |
| `.glass-card`    | Backdrop-blur — reserved for **over-content overlays only** (topology canvas, etc.) — not the default card style |
| `.badge-*`       | Status badges using canonical status hexes             |
| `.input-base`    | Standard input                                         |

## Theme Switching

Dark mode is toggled via the `useTheme` hook, which adds the `.dark` class to `<html>`. Default boot
is dark (matches Seed and Stem).

```tsx
import { useTheme } from "../hooks/useTheme";
const { theme, setTheme, actualTheme } = useTheme();
```

## Mobile Browser Chrome

`<meta name="theme-color" content="#3730a3" />` (NIAC indigo-700) — matches the brand identity
shown in the iOS/Android status bar.

## Accessibility

- All foreground/background pairs pass WCAG AA (4.5:1 for normal text, 3:1 for large text)
- Status meaning never relies on color alone — pair with icon/text
- `prefers-reduced-motion` respected
- Focus visible: 2px ring at `var(--color-brand-primary)`

## Related Files

- `src/index.css` — CSS variable definitions, `@theme` block
- `src/styles/theme.ts` — barrel export of token modules
- `src/styles/themeDeviceColors.ts` — device/link/protocol domain colors (orthogonal to brand)
- `_web/mustardseednetworks-com/src/index.css` — cross-product canonical source-of-truth
