# Operator Surface Alignment

NIAC keeps three operator surfaces by design:

- CLI: automation, CI, scripting, headless systems, support/debug commands.
- TUI: terminal-first live demos and operators who do not want a browser.
- Web UI: guided workflows, visualization, config editing, packet inspection.

Best-practice rule: implement capability in the backend/domain layer first,
expose a lean CLI command for automation, then provide richer TUI/Web UI flows
on top of the same backend capability.

## Current Web UI Route Alignment

The Web UI route table is centralized in `ui/src/pageRegistry.tsx`; navigation
groups in `ui/src/navGroups.ts` match those registered routes. Dynamic device
editor routes are registered explicitly in `ui/src/App.tsx`:

- `/`
- `/runtime`
- `/devices`
- `/device-config`
- `/device-config/new`
- `/device-config/:hostname`
- `/topology`
- `/automation`
- `/traffic`
- `/debug`
- `/packets`
- `/config-diff`
- `/walk-validator`
- `/library/walks`
- `/library/pcaps`

Legacy URLs redirect intentionally:

- `/templates` -> `/runtime`
- `/neighbors` -> `/topology`
- `/analysis` -> `/traffic`
- `/pcap-analyzer` -> `/packets`

The production UI build and unit test suite pass after the current alignment
work.

## TUI Improvement Backlog

Bubble Tea/Lipgloss remains the right TUI stack. Cobra is the CLI framework,
not the TUI framework.

Recommended next improvements:

1. Add a TUI command palette so every major action is discoverable without
   memorizing shortcuts.
2. Add a persistent footer/status legend that changes per panel and shows the
   valid keys for the current context.
3. Convert config editing panels from display-first overlays into task-focused
   flows: select device, edit interface, edit SNMP, preview diff, apply.
4. Add explicit unsaved-change tracking for TUI config mutations and a save or
   apply workflow instead of only mutating in-memory state.
5. Add TUI/API parity checks in tests so a Web UI or CLI feature cannot land
   without at least a TUI read/control path when it is an operator workflow.
6. Add visual regression snapshots for key TUI screens once the layout is more
   stable.
