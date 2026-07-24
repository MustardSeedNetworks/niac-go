# NIAC Web UI

The NIAC web UI is a React/TypeScript application embedded in the NIAC binary.
It uses the same HTTPS API and authoritative runtime state as the CLI and
daemon.

## Access

Start the daemon and open:

```text
https://localhost:8445
```

The daemon is HTTPS-only. A development installation uses a self-signed
certificate unless the operator installs a trusted local certificate.

When authentication is enabled, the UI asks for the bearer token in the active
tab. The token is kept in memory, is never placed in a URL or persistent
browser storage, and is cleared when the tab reloads or closes. Mutating
requests also use the per-session CSRF token returned by the daemon.

## Browser support

| Tier | Browsers | Release evidence |
| --- | --- | --- |
| First class | Chrome stable, Edge stable, Safari current | Critical journeys in the installed browsers |
| Engine CI | Playwright Chromium, WebKit, Firefox | Critical journeys on relevant pull requests |
| Compatibility | Firefox current | Independent-engine coverage and reproduced defect fixes |
| Best effort | Brave current | Pre-release smoke test with default Shields |

Playwright WebKit is automated coverage, not proof of Safari compatibility.
Release candidates must also be exercised in actual Safari.

The critical journey includes authentication, navigation, routed preflight,
start/stop/restart, live topology and statistics, SSE reconnection, offline
packet analysis, dialogs, clipboard actions, downloads, and narrow-width
layout.

## Primary workflows

- **Dashboard** — build, license, simulation, device, and traffic status.
- **Simulation** — choose a configuration and physical attachment, run
  preflight, and start or stop the runtime.
- **Running Devices** — inspect the authoritative state of the active devices.
- **Device Library** — create and edit saved device configurations.
- **Topology** — view authored and observed links and discovery data.
- **Alerts** — configure observable runtime notifications.
- **Fault Injection** — inject and clear supported interface or protocol
  faults and control PCAP replay.
- **Logs** — inspect the live structured log stream.
- **Packets** — inspect live packets or analyze an uploaded capture while no
  live capture is running.
- **Compare & Merge** — compare or merge configuration documents.
- **SNMP Walks** — validate, analyze, sanitize, and manage walk files.
- **License** — view the active local license and feature grants.

The canonical route registry is `ui/src/pageRegistry.tsx`; navigation groups
are defined in `ui/src/navGroups.ts`.

## Live updates

Simulation status, topology, statistics, logs, and packet streams use SSE or
WebSocket endpoints as appropriate. Clients reconnect interrupted event
streams and fetch authoritative state again rather than treating the last
event as durable truth.

Device tables use shared filtering, pagination, and virtual scrolling for
larger result sets. NIAC Free is limited to ten simulated devices. NIAC Pro
removes that tier soft cap; every path still enforces the absolute
1,000-device ceiling.

## Development

Frontend source lives in `ui/src`. The repository pins Node and npm in
`.nvmrc` and `ui/package.json`.

```bash
cd ui
npm ci
npm run lint
npm test
npm run build
```

`npm run build` writes directly to `internal/api/ui`. Do not copy or sync a
second build directory. A release build always uses `make build` so the
frontend and backend are built together and the embedded UI hash is injected
into `/__version`.

The UI uses:

- TypeScript with strict checking through `tsgo`;
- Biome for linting and formatting;
- Vitest and Testing Library for component behavior;
- Playwright for critical browser journeys;
- Tailwind CSS and the design tokens in `ui/src/index.css`;
- i18next with English and Spanish catalogs.

## Verification

For a UI or API contract change, run:

```bash
make fmt-check
make lint
make test
make test-e2e
make security
make build
```

Browser-channel CI runs installed Chrome and Edge for release candidates and
on its scheduled workflow. Actual Safari and Brave evidence is recorded during
release acceptance.

## Troubleshooting

### The page cannot connect

1. Confirm the daemon is running.
2. Verify `https://localhost:8445/__version`.
3. Accept or trust the development certificate only after confirming it is the
   expected NIAC endpoint.
4. Check the daemon log for listener or authorization errors.

### Authentication fails

1. Confirm the token was created by the current daemon configuration.
2. Re-enter it after a reload; tokens are intentionally not persisted.
3. Check for a structured authorization error in the browser network panel and
   daemon log.

### Live data stops

Confirm the simulation is still running, then inspect the affected SSE or
WebSocket request. The UI should reconnect automatically; a repeated failure
is a defect and should include the browser/version and daemon log.

### The UI does not match the installed version

Check `/__version`. `uiBuildHash` must be non-empty. If it is empty, the binary
was not built through the supported full-build pipeline.
