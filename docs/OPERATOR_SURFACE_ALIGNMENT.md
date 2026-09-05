# Operator Surface Alignment

NIAC keeps two operator surfaces by design:

- CLI: automation, CI, scripting, headless systems, support/debug commands.
- Web UI: guided workflows, visualization, config editing, packet inspection.

Best-practice rule: implement capability in the backend/domain layer first,
expose a lean CLI command for automation, then provide a richer Web UI flow
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

Pre-1.0 route aliases are not retained. Unknown or retired UI paths return to
the dashboard through the wildcard route; documentation and tests use only the
canonical paths above.

There is no automatic CLI/Web symmetry requirement. A new surface is added
only when a concrete operator workflow needs it and the product feature gate
accepts its build and maintenance cost.
