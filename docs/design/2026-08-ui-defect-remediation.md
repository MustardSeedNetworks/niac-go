# UI defect remediation — 19 defects, 8 waves

**Status:** in progress · **Owner-approved:** 2026-08-22
**Baseline:** origin/main @ 0.94.58 (`0710c026`), `make test` green
**Epic:** #1437 · **Issues:** #1418–#1436

Found by driving every sidebar item of the web UI against the live daemon on
CT304 (build 0.94.46), then re-verifying each defect against the source. All 19
sites were re-confirmed present on `origin/main` @ 0.94.58 before this plan was
written — none were fixed by the 50 commits between 0.94.46 and 0.94.58.

**Why the existing suite did not catch any of this:** `make test` and 23
Playwright specs pass while all 19 defects are live — including
`device-crud.spec.ts` and `device-editor.spec.ts`, for two flows that are
provably broken. Green tests are therefore _not_ the completion signal for this
programme. Every fix below lands a guard that is proven to fail against the
pre-fix code first.

**Owner decisions (2026-08-22):**

- CT304 may be redeployed for live verification; 0.94.46 + config backup kept as rollback.
- D19 native VLAN: demux key 0 = native, at most one per interface, coexists with N tagged sessions.
- Self-merge on genuinely green CI; never `--admin` past a red gate.
- Cut and validate a release once the programme completes.

---

Every claim below was re-verified against the source on 2026-08-22. Where the
sweep's original write-up was wrong, the correction is called out inline.

Repo: `~/Developer/MustardSeedNetworks/niac-go` · Report: DEFECTS.md · Evidence: CHECKLIST.md

---

## Corrections to the sweep report (verified during planning)

| # | Sweep said | Code says |
| --- | --- | --- |
| D8 | Fix is in `ConfirmModal.tsx`; "Escape does not dismiss" | Markup is in `ui/Modal.tsx:66-108`. Escape **is** wired (document-level, `closeOnEscape=true`). See **D18**. |
| D14 | Menu overflows its container | `Card`'s `backdrop-blur-xl` (`ui/Card.tsx:41`) creates a **stacking context**; the graph Card paints over the menu's `z-50`. |
| D15 | `fitView` never called | `fitView={true}` **is** set (`TopologyPage.tsx:871-872`). Cause is a `setTimeout(…,150)` race in `handleResetLayout` (`:338-351`). `handleRefresh` (`:534-549`) already fixed this with `requestAnimationFrame` and its comment documents the exact bug. |
| D16 | Server supplies the literal `"Unknown"` for IPs | Server **omits** the keys; the literal is set client-side at `PacketInspectorPage.tsx:169-170`. Server also emits packet fields **flat** (`src_mac`, `source_ip`, …) while the client reads a nested `headers.ethernet` / `headers.ipv4` object that is never sent. |
| D11 | "One deletion — no snake_case request structs remain" | True for `internal/api`, **false repo-wide**: `internal/drafttopology` still has **9** snake_case tags and is strict-decoded. Deleting the conversion without migrating them first breaks the wizard's visual topology editor. |
| D2 | One set of shipped templates | **Three** template trees exist. Deployed set is `internal/library/starter/` (md5-verified against CT304). It is the only one **not** validated by CI. |

**New defect found while planning — D18.**
`Modal.tsx:40-43` passes `onEscape: closeOnEscape ? onClose : undefined` into
`useFocusTrap`, whose effect deps are `[isActive, onEscape, autoFocus, restoreFocus]`.
Consumers pass inline arrows — e.g. `DebugConsolePage.tsx:317`
`onCancel={() => setShowClearConfirm(false)}` — so `onEscape` is a new identity
every render. The focus trap therefore **tears down and re-arms on every parent
render**, and each teardown runs `previousActiveElement.current.focus()`
(`restoreFocus`), yanking focus out of the dialog. On a continuously
re-rendering page (Logs at Trace level) focus thrashes many times per second.
This is the likely cause of the Escape miss observed during the sweep.
_Fix: `useCallback` the handlers, or hold `onEscape` in a ref inside the hook._

---

## Preconditions (before any fix branch)

1. `git -C ~/Developer/MustardSeedNetworks/niac-go fetch && git switch -c <branch> origin/main` — never commit to `main`.
2. Establish a green baseline: `make lint && make fmt-check && make test`. Record the result. Every wave is judged
   against this, not against a hopeful clean slate.
3. Note the test contract: `make test` = `go test -race -coverprofile` over all non-ui packages **plus** Vitest.
   `make test-e2e` = full `make build` then Playwright (`ui/e2e/*.spec.ts`, 23 specs) against a **real built
   daemon** — E2E never mocks the backend, which is why contract bugs must be caught there or in Go handler tests,
   not in Vitest.
4. Vitest coverage gates: lines 75 / branches 67 / functions 45 / statements 67. New guard tests must not drop these.

---

## Wave 1 — the two unblockers

### F1 · D11 — finish the ADR-0007 migration on the client

This is not a patch; it is the completion of an **accepted** architectural
decision. `docs/adr/0007-json-wire-casing-convention.md` (Accepted 2026-06-16)
mandates _"Every field NIAC's API emits or accepts is camelCase. There are no
wire-level snake_case exceptions."_ The server side is done — my audit found
**zero** snake_case `json:` tags on any request struct in `internal/api` (the ADR
recorded 74 at the start). The client was never migrated.

Order matters:

1. **Migrate `internal/drafttopology` to camelCase** — 9 tags, strict-decoded at
   `internal/api/handlers_draft_topology.go:35`:
   `admin_status, fdb_only, in_utilization, mac_suffix, native_vlan, oper_status, out_utilization, profile_role, sys_object_id`.
   The client already _names_ these fields in camelCase
   (`library-client.ts:81-100`: `macSuffix`, `sysObjectId`, `profileRole`,
   `adminStatus`, `operStatus`) — they only match today **because**
   `toSnakeCase` converts them. This package was outside the ADR's
   `internal/api` audit scope, which is why it was missed.
2. **Remove the outbound conversion** at both call sites:
   - `ui/src/api/requestCore.ts:366` — `body: JSON.stringify(toSnakeCase(payload))`
   - `ui/src/api/requestUpload.ts:89` — `xhr.send(JSON.stringify(toSnakeCase(payload)))`
   `requestJsonCamelCase` (`requestCore.ts:372`) is already byte-for-byte
   `requestJson` minus the conversion; collapse the two rather than leaving a
   duplicate. Keep `toCamelCase` — it is still needed on **responses**.
3. **Keep or drop `toSnakeCase`** based on remaining usages after step 2. If
   unused, delete it and its unit tests; if still used, leave it exported.

**Closes:** device create · device clone · device update · inject error · PCAP
replay · save alerts — plus the latent ones never exercised
(`templates/use` → `newConfigName`/`templateName`; `synthesize-walk` →
`interfaceCount`; `walk/validate` → `autoFix`).

**Test landmines:** none, under _this_ direction. `library-client.test.ts`
(`'creates template drafts without flattening resources in the browser'`)
asserts a **camelCase** body for `createScenarioDraftFromTemplate` — which
bypasses `requestJson` entirely today. Moving everything to camelCase keeps it
green. (An earlier audit flagged it as a landmine only because it assumed the
opposite fix direction — standardising on snake_case. That would contradict
ADR-0007 and must not be chosen.)

**Guard:** a Go handler test per mutating endpoint that posts the **client's real
payload shape** and asserts 2xx. Constructing the Go struct in the test cannot
catch this class — the bug lives in the JSON key names, so the fixture must be
raw JSON.

### F2 · D8 (+ D18) — make dialogs clickable

`ui/src/ui/Modal.tsx:78-85` — the `role="dialog"` div has **no positioning
class**. Its sibling scrim is `absolute`, and positioned elements paint above
non-positioned ones regardless of DOM order, so the scrim covers the dialog.

- Add `relative z-10` (or equivalent) to the dialog div in `Modal.tsx`. This one
  change covers `ConfirmModal` and every one of its **8 production consumers**:
  `ErrorInjectionPanel`, `config/MergeControls`, `DebugConsolePage`,
  `WalkValidatorPage`, `DeviceListPage`, `RuntimeControlPage`,
  `LibraryFilesPage`, `DeviceEditorPage`.
- Apply the same to `ui/src/components/device-list/CloneDeviceModal.tsx:41`,
  which reimplements the pattern independently.
- `TemplatePreviewModal.tsx:111` and `MergeControls`' `MergePreviewModal:356`
  already set `relative` — leave them; they are the proof the fix is right.
- **D18:** stabilise the handler identity (`useCallback` at consumers, or ref the
  callback inside `useFocusTrap`) so the trap stops re-arming every render.

**Guard:** a Playwright assertion —
`elementFromPoint(centre of confirm button) === that button` — for at least one
ConfirmModal instance and for `CloneDeviceModal`. A DOM-presence test passes
today and would keep passing; only hit-testing catches this.

**Coverage note:** no `ConfirmModal.test.tsx` exists. The guard is new code.

---

## Wave 2 — stop the types lying about the wire

### F3 · D6 — nil slices must marshal as `[]`

`internal/fabric/types.go:130-149`:

```go
type Topology struct {
    Binding    CompiledBinding `json:"binding"`
    Networks   []Network       `json:"networks"`
    Interfaces []Interface     `json:"interfaces"`
    Routes     []Route         `json:"routes"`
    DHCPScopes []DHCPScope     `json:"dhcpScopes"`
}
type Report struct { Safe bool; Topology Topology; Diagnostics []Diagnostic }
```

All five are populated only via `append` in `internal/fabric/compiler.go` and
`compiler_devices.go`; `Compile()` never seeds them, so a config with no
networks leaves them nil → JSON `null`. **`omitempty` does not fix this** — a nil
slice still marshals as `null`; initialise to `[]T{}` in `Compile`.

**Blast radius is one type, both endpoints:** `internal/api/server.go:377-384`
`SimulationFabricStatus` embeds the same `fabric.Topology`, so `GET /api/v1/simulation`
carries the identical exposure and is fixed by the same change.

Client side: `ui/src/api/fabric-types.ts:53-60` must stop asserting these
non-nullable, and `PreflightStep.tsx:147` must guard.

**Coverage gap:** both existing preflight fixtures
(`PreflightStep.test.tsx:25-28`, `client.preflight.test.ts:31-34`) use `[]`,
never `null` — which is exactly why this shipped. **Guard:** a Go marshal test
asserting `[]` for an empty compile, plus a UI fixture with `null` topology.

### F4 · D10 — unwrap the device response

`ui/src/components/device-editor/useDeviceEditor.ts:164` waits for
`fetchedDevice?.device`; `handleDeviceGet` writes the device **flat**. The type
`ui/src/api/device-config-types.ts:163` asserts the wrapper, so TS cannot catch it.

**Coverage gap:** `useDeviceEditor.test.tsx` mocks the wrapped shape, but
`useParams` is pinned to `{hostname:'new'}` for the whole file and the hook
short-circuits that case — so **the edit-load path has no test at all**. Guard is new.

### F5 · D13 — stop reporting a frozen device count

`internal/daemon/daemon.go:1119` `status.DeviceCount = sim.cfg.DeviceCount()`.
`sim.cfg` is set once at session start (`daemon.go:706`) and **never reassigned**
— the daemon has no "replace config on a running session" API. Device mutations
call `api.Server.saveConfig` → `replaceConfig` (`config_state.go:132-140`), which
updates the _API layer's_ state only. The `ApplyConfig` hook (`server.go:313`)
that could bridge them is **never wired in production** — assigned only in tests,
and `cmd/niac/runtime_services_test.go::TestRuntimeServicesApplyConfigNil`
asserts it is nil at runtime.

Two options: wire `ApplyConfig` end-to-end, or have `simulationStatus` read the
same live config the correct endpoints use. **Prefer the latter** — smaller, and
it removes a dead hook rather than activating one.

### Strategic — generate the TS types from the Go structs

D6, D10 and D13 are one failure mode: a TypeScript type asserting what the wire
does not deliver. Patching three sites leaves the mode intact. Generating
`ui/src/api/*-types.ts` from the Go structs turns each of these into a compile
error. This is the change that makes "zero" hold instead of decay.

---

## Wave 3 — data the product already has and discards

### F6 · D9 — protocols column and filter

`ui/src/components/device-list/deviceFilters.ts:3-32` re-derives protocols from
sub-objects (`snmpAgent`, `lldp`, …) that the list response does not carry.
Verified server-side: `handleDeviceList` (`internal/api/devices.go:68`) only
includes them when `?details=true`, and `fetchConfigDevices()` sends no params.
Two valid fixes — read the `protocols` array the API already returns (preferred),
or request `?details=true` (heavier payload for 500+ device configs).
**Coverage:** no `deviceFilters.test.ts` exists at all. Guard must assert against
the **summary** shape, not a fixture with `snmpAgent` populated.

### F7 · D12 — DECIDED (owner, 2026-08-22): derive segments from per-device VLAN

Owner's rule: _"multiple scenarios each on their own tag, or an untagged
scenario, or tagged as well — either should work, just like anything else in
the networking world."_ Applied to Segments that settles option (a).

Two different things are called "VLAN" in NIAC; the decision touches both:

| | What it is | Status |
| --- | --- | --- |
| **Physical VLAN** (`Binding.AccessVLAN`) | the tag a whole _scenario_ rides on the wire | works — 6 scenarios verified on tags 200-205 |
| **Internal VLAN** (`Segment.Tag`, per-device `vlan`) | VLANs _inside_ one simulated network (mgmt 200, data 210, wifi 220, servers 240) | **broken — this is D12** |

`Segment.Tag == 0` is `config.UntaggedTag` — the **native VLAN** sentinel
(`internal/config/types.go:164-171`). Today it is misused as a catch-all: with no
explicit `segments:` block, `NormalizedSegments()` dumps every device into tag 0,
so a config the daemon knows is split `{200:40, 210:7, 240:6, None:3}` renders as
one "Untagged" bucket.

**Fix:** derive segments from the per-device VLAN the config already resolves
(the same source `/api/v1/config/devices` uses for its `vlan` field). Keep tag 0
as a _real_ native/untagged bucket for genuinely untagged devices — both must
work; neither is a dumping ground.

**Test landmines — these assert the catch-all as correct and must be rewritten:**

- `internal/config/segments_test.go::TestNormalizedSegmentsBackwardCompat`
- `internal/api/handlers_segments_test.go::TestHandleSegmentsFlatConfigIsUntagged`

Rewrite them to assert: a config with per-device VLANs yields one segment per
VLAN; a config with genuinely untagged devices yields a tag-0 segment; a config
with both yields both.

### F7b · D19 (NEW) — an untagged scenario cannot run alongside tagged ones

Direct consequence of the owner's rule, and a real gap. On one interface NIAC
supports N trunk-tagged scenarios, but **exactly one** untagged scenario with
nothing beside it — `internal/daemon/session_registry.go:37`:

```go
if active.Binding.Mode != fabric.ModeTrunk || binding.Mode != fabric.ModeTrunk {
    return fmt.Errorf("%w: %s", ErrInterfaceInUse, binding.Interface)
}
```

If either session is `access` or `direct`, the second is refused outright. But a
real trunk port carries a **native VLAN** — untagged frames alongside tagged
ones. NIAC already models exactly that for the devices it _simulates_
(`config.TrunkPort.NativeVLAN`, validated at `internal/config/validator.go:669`);
its own host attachment has no equivalent — `fabric.Binding`
(`internal/fabric/types.go:61-67`) carries only `Mode` + `AccessVLAN`.

Today untagged frames arriving on the trunk are **counted and dropped**
(`internal/daemon/trunk_capture.go:54-57`, `recordUntagged()`). The sweep saw
this live on CT304: `drops.untagged = 67`.

**Fix shape (verified to fit the existing structures):** the capture demux is
already `sessions map[uint16]*trunkSessionTransport`
(`trunk_capture.go:72`), keyed by tag. Reserve key **0** for a native session —
consistent with `config.UntaggedTag = 0` — route the frames `recordUntagged`
currently discards to `sessions[0]` when one is registered, and relax
`validateReplacement` so a native session may coexist with trunk sessions
(still at most one native, exactly like a real trunk port).

**Size:** the largest single item in this plan — it touches the binding model,
the session registry, the capture demux, and preflight validation. Treat it as
its own PR with its own design note, not a drive-by.

### F8 · D16 — packet decode

Two independent faults, both code-confirmed:

1. **Shape mismatch.** `internal/api/sse/packet_observer.go:84-135` emits a
   **flat** map (`src_mac`, `dst_mac`, `source_ip`, `dest_ip`, `protocol`,
   `raw_data`…). `ui/src/utils/protocol-layers.ts:107` reads nested
   `headers.ethernet.srcMac`. `PacketInspectorPage.tsx:183` passes
   `incoming.headers`, a key the live stream never sends — so the MAC row shows
   "(not parsed)" for **every** packet, not only STP. Note this also violates
   ADR-0007 (SSE payloads are explicitly in scope) — fold the key renaming into F1's convention.
2. **No L2 awareness.** `enrichNetworkLayer`/`enrichTransportLayer` only test
   `IPv4/IPv6/ARP` and `TCP/UDP/ICMPv4/ICMPv6`. There is no `Dot1Q`, `LLC` or
   `SNAP` branch, so a tagged STP BPDU keeps the `"Unknown"` default.
3. The literal `"Unknown"` for IPs comes from `PacketInspectorPage.tsx:169-170`,
   client-side — not the server.

**Open sub-item:** the sweep observed a phantom _IPv4_ layer render, which
`buildIpLayer` should not produce when `headers` is undefined. Trace this during
implementation rather than assuming — it implies a second population path.

---

## Wave 4 — shipped content (D2)

Three parallel template trees exist:

| Tree | Files | Validated in CI? | Deployed? |
| --- | --- | --- | --- |
| `internal/library/starter/` | 8 | **No** | **Yes** — md5-identical to CT304 |
| `internal/templates/builtin/` | 8 | Yes (`internal/templates/templates_test.go::TestTemplateConfigValidity`) | No |
| `cmd/niac/templates/` | 7 | Yes (`internal/api/templates/templates_test.go::TestShippedTemplatesAreValid`, full validator) | No |

`starter/` is bootstrapped into the on-disk library on first run
(`internal/library/bootstrap.go:16`, `//go:embed starter/*.yaml`) and is what the
wizard lists. **It is the only tree with no validation test, and the only broken one.**
Proof of divergence — the same file in each tree:

```text
builtin/small-office.yaml   dhcp: subnet_mask / pool_start / pool_end / router   ← current schema
starter/small-office.yaml   dhcp: enabled / pools / range_start                  ← obsolete
```

`builtin/` was migrated when `converter.DhcpServer` changed; `starter/` was not.

**Fix:** collapse to one tree (pre-v1.0.0 — no compat burden, so delete the
duplicates rather than syncing three copies), then extend
`TestShippedTemplatesAreValid`'s full-validator walk to cover it.

**Also fix the false-clean signal:** `internal/library/list.go` sets `Valid: true`
whenever `countDevices` can `yaml.Unmarshal` into `struct{Devices []map[string]any}`
— a syntax check that never calls `config.Load` or any validator. That is why the
library advertises eight healthy templates that cannot load.

**Guard:** `niac --dry-run` (or the full validator) over the shipped template dir
in CI — it catches all eight today.

---

## Wave 5 — diagnostics

- **F10 · D3.** `details[]` is already parsed correctly into `ApiError.details`
  (`ui/src/api/errors.ts`, `requestCore.ts:211-227`). It is dropped in the shared
  toast path: `getErrorMessage` (`utils/format.ts:146-154`) returns `err.message`
  only, `useErrorToast.ts` never reads `.details`, `Notification`
  (`stores/ui-store.ts:74-81`) has no field for it, and `ToastContainer.tsx:21-41`
  has nowhere to render it. Five pages already read `.details` bespoke
  (`BpfFilterBar.tsx:20`, `DevicesPage.tsx:134`, …) — proof the data is reachable.
  Fix all four layers once and delete the bespoke workarounds.
- **F11 · D5.** `internal/api/handlers_simulation.go:126` collapses the error and
  passes `nil` details. Return structured details, as `/api/v1/library/drafts`
  already does. Scope: the **error** path only — `safe:true`/`safe:false` both
  report correctly today.
- **F12 · D7.** `ui/src/utils/error-reporter.ts:37` POSTs `/api/v1/client-errors`;
  the server registers `/api/v1/errors` (`routes.go:361`). Implement the endpoint
  or repoint the client. Note the `.catch()` cannot help — a 404 is a resolved fetch.

## Wave 6 — layout and reachability

- **F13 · D1.** `HelpDrawer.tsx:183` `<nav className="flex gap-tight -mb-px">` —
  add wrap or `overflow-x-auto` with scroll-into-view on activation.
  **Coverage:** nothing asserts all 8 tabs are reachable (`HelpDrawer.about.test.tsx`
  only checks the version badge; `e2e/help-drawer.spec.ts` only opens/closes).
- **F14 · D14.** Portal the `ActionsMenu` dropdown out of the header `Card`, or
  give the header Card an explicit higher stacking order. The `z-50` on the menu
  is already correct and cannot win across the `backdrop-blur` stacking contexts.
- **F15 · D15.** Port `handleResetLayout` (`:338-351`) and the device-set-change
  effect (`:436-453`) onto the `requestAnimationFrame`-after-settled pattern that
  `handleRefresh` (`:534-549`) already uses — its own comment documents why the
  fixed `setTimeout` is wrong.

## Wave 7 — polish

- **F16 · D4.** `DraftBehaviorComposer.tsx:164` — state the interface
  precondition in the empty state; gate Save on ≥1 timeline (`[].every()` is `true`).
- **F17 · D17.** `SettingsDrawer.tsx:470-471` — derive React/TypeScript versions
  from the build instead of the literals `"19.2"`/`"5.9"` (actual: 19.2.8 / **7.0.2**).
  Settle the product name: `internal/i18n/locales/en/settings.json:10` says
  "Network Injection & Analysis Console"; help content and `es/common.json` say
  "Network In A Can".

## Wave 8 — close the coverage gap

Zero _known_ defects != zero defects. These surfaces were never driven and must be
swept before the claim holds: wizard drag-to-connect and in-wizard Add device
(**note: this is exactly the path F1 step 1 protects — sweep it immediately after
Wave 1**), the "Start empty" path, real timeline creation, Walk Analyzer
"Capture from device", Packets Follow-Stream / Colors / capture-side BPF Apply,
per-page Help buttons, the Dashboard "Error Injection" quick action, a
light-theme pass, and mobile/tablet widths.

---

## Sequencing summary

| Wave | Closes | Gate to advance |
| --- | --- | --- |
| 1 | D11, D8, D18 | `make test` green; contract + hit-test guards land; **re-drive the wizard topology editor** |
| 2 | D6, D10, D13 | marshal test proves `[]`; edit-load test exists |
| 3 | D9, D12, D16, **D19** | protocols visible; segments split by VLAN; a native + tagged scenario run together; tagged BPDU decodes |
| 4 | D2 | CI validates the deployed template tree |
| 5 | D3, D5, D7 | a failing save shows the server's reason |
| 6 | D1, D14, D15 | all 8 help tabs and all 4 exports mouse-reachable |
| 7 | D4, D17 | — |
| 8 | unknown | second sweep over untested surface |

**Standing constraints:** feature branch + conventional commits + PR (`main` is
protected); `make lint` / `make fmt-check` / `make test` clean per PR; no
`//nolint` or `biome-ignore` without asking; fix root causes, no suppressions.

---

## Closeout — 2026-08-23, released as v0.94.61

All nineteen defects are fixed, merged and verified against a running daemon on
CT304. Eight further defects were found _by_ the verification pass and were
fixed the same way; every one of them was invisible to the test suite before its
guard landed.

### Verified live on the released 0.94.60 build

| Defect | Evidence on CT304 |
| --- | --- |
| D1 | 8 help tabs, 0 blocked by hit-test |
| D2 | 7 scenario packs offered, each with version/device/link metadata |
| D6 | `networks`/`interfaces`/`routes`/`dhcpScopes` all lists |
| D7 | 0 console errors across all 16 routes |
| D8 | both dialog buttons hit-test to themselves; mouse Cancel closes |
| D9 | first device reports `SNMP, DHCP, DNS, LLDP, CDP` |
| D10 | editor populates `LAB-EDGE-R1` + MAC |
| D11 | "Alert configuration saved"; in-wizard Add device, 0 failed requests |
| D12 | segments group as `(0,3) (200,80) (210,36) (240,28)` |
| D13 | runtime 147 == simulation 147 |
| D14 | all 4 export items mouse-reachable |
| D15 | graph centred — `nodesCenterX` 964 == `paneCenterX` 964 |
| D16 | STP/LLDP/CDP named; no `Unknown` |
| D17 | TypeScript 7.0.2, React 19.2.8 |
| D3, D5 | a failing preflight lists `devices[0].snmp_agent.community: SNMPv1/v2c requires an explicit community` and the same for `devices[1]` |
| D4 | Behaviors names why its button is disabled: "Add at least one interface to a device before scheduling behaviors" |
| D18 | Escape closes the stop-simulation dialog (completed by #1465) |
| D19 | see below |

### D19 acceptance

`--attachment-policy eth0=trunk:200,...,299 --attachment-policy eth0=access:210`
is accepted, and the trunk capture reports:

```text
sessionVlans: [0, 200, 201, 202, 203, 204, 205]
drops: {untagged: 0, unapproved: 357, overrun: 0}
```

Demux slot 0 is the native session, live alongside six tagged ones, each with
its own independently climbing RX. Untagged ARP frames injected on the host end
of the veth were confirmed arriving by `tcpdump` and produced **zero** untagged
drops, proving they were delivered to the native session rather than discarded
— that counter read 67 before this work.

### Found by the verification pass

| Issue | Defect |
| --- | --- |
| #1457 | every packet reported arrival time `0001-01-01T00:00:00Z` |
| #1458 | the protocol tree fabricated an IPv4 layer on LLC frames |
| #1460 | every saved config gained `snmp_agent: {}`, so wizard drafts failed their own validation |
| #1461 | preflight reported "N configuration errors found" without naming any |
| #1463 | one policy per interface, so a native scenario could never be approved |
| #1465 | Escape never closed any modal |
| #1467 | two preflight paths still returned `null` arrays |
| #1472 | the wizard discarded the details #1461 had just made the server send |

Three were residuals of earlier fixes in this same programme — #1458 (D16
guarded only the first of two branches), #1467 (D6 seeded only two of four
construction sites), and #1472 (D3 threaded details through the toast path while
the wizard renders an inline banner) — and #1465 was a second, independent cause
behind D18. The recurring shape is a fix that covers one branch, one
construction site, or one surface, and reads as finished.
That is the argument for verifying against a running system rather than a green
suite: **`make test` and 23 Playwright specs passed while all nineteen original
defects were live**, because `ui/e2e/device-editor.spec.ts` stubs
`**/api/v1/**` and fulfils 201 unconditionally.

### Deferred, with rationale

- **#1462** — `Device.SNMPConfig` is value-typed while every sibling protocol
  config is a pointer, so "no SNMP" is unrepresentable. #1460 was fixed at the
  writer, which is correct and local; the pointer migration changes load and
  marshal semantics for every config, template and walk scenario and needs its
  own red-first change.
- **#1469** — release binaries build with Go 1.27.0 while `go.mod` declares
  1.26.6. Owner call: the same container pin exists in seed and stem, and the
  fleet rule is to bump all three in lockstep.

### Wave 8 sweep

All 16 registered routes drove clean: HTTP 200, no error boundary, no console
errors, no failed API calls. The wizard was driven end to end through
Start-empty → Add device → Protocols → Review → Connection → preflight → start,
which is the path D11 would have broken. That single run surfaced four of
the seven new defects above: the invalid draft, the unnamed preflight errors,
the unapprovable native policy, and the remaining null arrays.
