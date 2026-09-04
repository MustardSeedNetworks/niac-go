# NIAC Design System

Keeps styling consistent and theme-aware. Use the centralized tokens instead of
raw values.

## Token Architecture (read first)

Three tiers, **one** source of truth for values, **one** derivation direction:

```text
Primitive   Tailwind's built-in palette (indigo-600 = #4f46e5)   ← never referenced directly in app code
   ↓ alias
Semantic    index.css @theme + :root/.dark                       ← THE source of truth for VALUES
            brand-*, status-*, surface-*, text-*, module-*,
            device-*, link-*, proto-*, syntax-*, chart-1..10,
            log-*, scrim, knob, on-brand/on-danger/on-info
   ↓ alias
Component   index.css @layer components + the TS class-token
            objects in styles/                                   ← consume semantic tokens
```

**Two invariants (enforced by `scripts/check-token-discipline.sh`):**

1. **Values flow one direction** — defined once in `index.css`, everything else
   references them. Never hand-copy a hex sideways into a `.ts`/`.tsx` file.
2. **App code names only semantic / component tokens** — never a primitive
   palette utility (`bg-gray-500`, `text-pink-400`) and never a raw hex.

**Picking the right token:**

- Neutral chrome (backgrounds, borders, muted text) → `surface-*` / `border-*` / `text-*`.
- Meaning (ok / warn / error / info) → `status-*`.
- Filled-surface foreground → `on-brand` / `on-danger` / `on-info`; black/white → `scrim` / `knob`.
- Domain categoricals → the dedicated ramps: `device-*` (topology nodes),
  `link-*` (speeds), `proto-*` (protocols), `chart-1..10` (generic categorical),
  `module-*` (the 5 feature modules), `syntax-*` (YAML/code editor).

**Charts / SVG / inline styles / CodeMirror:** consume the CSS variables directly —
`fill="var(--color-device-router)"`, `color: 'var(--color-syntax-keyword)'`. They
flip light↔dark via the cascade. NIAC has no `<canvas>` drawing, so (unlike seed)
it needs no JS token-reader; the topology graph is SVG.

**Brand:** NIAC's anchor is **indigo** `#4f46e5` (`niac-500`). The five feature
modules have their own accents (`--color-module-{topology,protocols,analyze,inject,templates}`).
