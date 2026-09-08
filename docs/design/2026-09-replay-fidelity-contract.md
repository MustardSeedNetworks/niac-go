# Design: the walk replay substitution contract

**Issue:** #1901 (Phase 1f, row F0)
**Author:** Kris Armstrong
**Status:** Accepted
**Date:** 2026-09-08

## Problem

NIAC's charter is to replay an authored or captured network to an NMS. When a
device is backed by a walk file, `Agent.LoadWalkFile` deliberately does **not**
reproduce that capture byte for byte: some OIDs are replaced by what the
scenario authored, some are answered from the running agent's own counters, and
some are dropped in favour of the topology the scenario declares.

Those decisions were correct and were spread across five predicates and six
post-load refresh functions. Nothing wrote them down in one place, nothing
tested that everything _else_ arrives unchanged, and a fidelity harness would
have had to re-derive the same prefix lists — and drift from them.

So "fidelity" needs a precise definition:

> Every OID the wire returns is either byte-identical to the source walk, or
> accounted for by one named substitution below.

## The contract

`internal/protocols/snmp/walk_contract.go` is the single source of the prefix
lists. `Agent.WalkContract()` builds a `WalkContract` for one device;
`Classify(oid)` returns the bucket that covers it, and `DropsFromWalk(oid)` is
the narrower question `LoadWalkFile` asks. The loader and the fidelity harness
(row F1a) both go through it, so they cannot disagree about what a difference
means.

### Buckets

| Bucket | Meaning |
| --- | --- |
| `kept` | The walk's OID, type and value are served unchanged. The default, and the point of the exercise. |
| `authored` | Replaced by what the scenario authored. |
| `live` | Answered from this run's counters, not the capture's frozen ones. |
| `topology` | Dropped in favour of neighbours synthesized from `trunk_ports`. |
| `normalized` | The source line named its OID symbolically and was rewritten to numeric form on parse. Decided from the source text, so `Classify` never returns it. |
| `agent_added` | On the wire but not in the source: MIB-II the agent synthesizes for every device. Only the harness sees both sides, so `Classify` never returns it. |

### The five signed substitutions

Owner sign-off 2026-09-05, all five accepted:

> sysName from config; sysDescr/Contact/Location only when authored;
> snmp group + ip/icmp/tcp/udp/egp live; LLDP/CDP/FDB neighbours dropped
> only under `trunk_ports`; authored interface rows override the walk.
> Everything else must arrive byte-identical.

| # | Source OIDs | Bucket | Condition | Mechanism |
| --- | --- | --- | --- | --- |
| 1 | `sysName.0` (`1.3.6.1.2.1.1.5.0`) | `authored` | always | skipped at load |
| 2 | `sysDescr.0`, `sysContact.0`, `sysLocation.0` | `authored` | only when the scenario sets that field | skipped at load |
| 3 | `1.3.6.1.2.1.11` (snmp group) and the `ip`/`icmp`/`tcp`/`udp`/`egp` subtrees | `live` | always | skipped at load, then re-registered live |
| 4 | LLDP remote systems, CDP cache, `dot1dTpFdbTable`, `dot1qFdbTable`, `dot1qTpFdbTable` | `topology` | only when the device declares `trunk_ports` | skipped at load |
| 5 | `ifTable`/`ifXTable` rows for interfaces the scenario authored | `authored` (configuration columns) or `live` (counters) | only at an authored `ifIndex` | **loaded, then overwritten** after the load |

Substitution 5 splits within a single row. Configuration columns —
`ifSpeed`, `ifHighSpeed`, `ifMtu`, `ifType`, `ifConnectorPresent`,
`ifAdminStatus`, `ifOperStatus`, `ifAlias`, `dot3StatsDuplexStatus` — come from
the scenario. The octet, packet, discard and error counters (32- and 64-bit)
become dynamic and report this run's traffic.

Two consequences worth stating because they are easy to assume the other way:

- **A row nobody authored keeps the capture's static counters.** The interface
  substitution is scoped to authored `ifIndex`es, so a walk of a 48-port switch
  used by a device that authors two interfaces serves 46 frozen rows.
- **`ifDescr` and `ifName` are never substituted.** The walk owns the IF-MIB
  indexes, and those two columns are how an authored interface is matched to a
  row in the first place.

### Why substitution 5 is not a load-time skip

`Classify` needs the `ifIndex` each authored interface holds, which is resolved
from `ifDescr`/`ifName` in the MIB as it stands. The agent synthesizes an
`ifTable` at construction, so indexes exist before any walk is loaded — but a
walk can renumber them. A contract used to judge what reached the wire must
therefore be built _after_ the load. `DropsFromWalk` is deliberately narrower
than `Classify != kept`: it answers only the index-independent substitutions
(1–4), which is all the loader can decide while it is still reading the file.

## Found during F0, unsigned

Auditing the writers that run after the load loop turned up three places that
overwrite or delete OIDs a walk can carry, outside the five substitutions the
owner signed. They are **not** in the classifier: row F1a will measure them as
unclassified rows, which is the honest outcome, and the owner rules on each.

1. **`dot1dTpPortTable` (`1.3.6.1.2.1.17.4.4`) is overwritten for every bridge
   port the walk itself defines.** `refreshBridgePortCounters` runs after every
   load and calls `registerDot1dTpPortEntry` for each port with a
   `dot1dBasePortIfIndex` row, setting `dot1dTpPort`, `dot1dTpPortMaxInfo`
   (the default MTU) and `dot1dTpPortInDiscards` (0), and making
   `dot1dTpPortInFrames`/`OutFrames` dynamic. Substitution 4 covers
   `dot1dTpFdbTable` at `.17.4.3`, not this table — and unlike substitution 4
   this happens **whether or not the device declares `trunk_ports`**.
2. **Physical addresses are rewritten from the authored MAC.**
   `refreshAuthoredPhysicalIdentity` replaces every `ifPhysAddress.*` that holds
   a real address, plus `dot1dBaseBridgeAddress` and `lldpLocChassisID`, and
   rewrites `entPhysicalSerialNumber.*`. The reasoning is the same as sysName —
   two devices sharing one walk must not collide — but identity substitution was
   signed for the `system` group only, and this is also independent of
   `trunk_ports`.
3. **`lldpLocPortTable` is deleted and rebuilt.**
   `refreshAuthoredDiscoveryMIBs` clears `lldpLocPortTable` along with the two
   neighbour tables. It is inside the `trunk_ports` gate, so it is narrower than
   (1) and (2), but the signed substitution names remote/CDP/FDB — the LLDP
   _local_ port table is a fourth prefix, and the classifier calls it `kept`
   because the load does not drop it — the refresh afterwards does.

Measured on a walk carrying `dot1dTpPortMaxInfo.1 = 9999`,
`ifPhysAddress.1 = AA BB CC DD EE FF` and a captured `lldpLocPortDesc.1`,
loaded by a device whose authored MAC is `00:11:22:33:44:55`:

| OID | Walk says | Served, no `trunk_ports` | Served, with `trunk_ports` |
| --- | --- | --- | --- |
| `1.3.6.1.2.1.17.4.4.1.2.1` (`dot1dTpPortMaxInfo`) | 9999 | 1500 | 1500 |
| `1.3.6.1.2.1.2.2.1.6.1` (`ifPhysAddress`) | `aa:bb:cc:dd:ee:ff` | `00:11:22:33:44:55` | `00:11:22:33:44:55` |
| `1.0.8802.1.1.2.1.3.7.1.3.1` (`lldpLocPortDesc`) | `captured-local-port` | `captured-local-port` | absent |

`initializeBridgeMIB` was audited and is clean: it returns early when the walk
supplied `dot1dBaseNumPorts`, so it only ever adds BRIDGE-MIB to a device that
had none (`agent_added`). `registerConfiguredRoutes` writes `ipRoute*`, which is
already inside substitution 3's `ip` subtree.

## What the harness measures (row F1a)

`TestWalkReplayFidelity` (`internal/protocols/snmp/fidelity_test.go`) loads each
walk into a fresh agent, sweeps the result twice through the agent's own PDU
path — a GET-NEXT chain and a GET-BULK sweep — and compares what comes back to
the **parsed source**, never to an exported walk file: the two formatters
disagree (`FormatWalkEntries` preserves Hex-STRING, `ExportToWalkFile` flattens
every OctetString to STRING), so an export-based diff would report type changes
that are the exporter's.

Every source OID must be byte-identical or land in exactly one contract bucket.
`NIAC_FIDELITY_REPORT_DIR` writes a per-walk JSON report; `NIAC_WALK_CORPUS`
points the sweep at the off-repo corpus instead of the shipped set.

### The shipped eighteen

| | |
| --- | --- |
| kept | 65,744 |
| substituted (by a signed bucket) | 3,591 |
| unclassified | 424 |
| dropped, type-changed | 0 |
| GET-NEXT / GET-BULK disagreements | 0 |
| ordering breaks | 0 |

All 424 unclassified rows fall in seven columns, and every one is an audit
finding above rather than a new defect class:

| Column | Rows | Finding |
| --- | --- | --- |
| `1.3.6.1.2.1.2.2.1.6` (ifPhysAddress) | 194 | 2 |
| `1.3.6.1.2.1.17.4.4.1.3/.4/.5` (dot1dTpPort) | 222 | 1 |
| `1.3.6.1.2.1.47.1.1.1.1.11` (entPhysicalSerialNumber) | 4 | 2 |
| `1.3.6.1.2.1.17.1.1.0` (dot1dBaseBridgeAddress) | 3 | 2 |
| `1.0.8802.1.1.2.1.3.2.0` (lldpLocChassisId) | 1 | 2 |

### The 745-walk corpus

Run on the Mac against `niac-demo-catalog/walks/sanitized`:

| | |
| --- | --- |
| walks | 745 |
| kept | 23,272,081 |
| substituted | 590,049 |
| unclassified | 52,771 across 698 walks |
| dropped, type-changed | 0 |
| GET-NEXT / GET-BULK disagreements | 0 |
| GET-NEXT returned OIDs out of order | **4 walks** |

The unclassified columns are the same three findings at scale — ifPhysAddress
35,118, the dot1dTpPort columns 13,379, entPhysicalSerialNumber 3,074,
dot1dBaseBridgeAddress 526, the two lldpLoc chassis columns 547, and a tail
including `ipNetToMediaPhysAddress`, which is another address rewrite.

The four ordering breaks are all Cisco Nexus walks, and the OIDs in them are
**symbolic** (`SNMPv2-MIB::sysORDescr.3`, `IP-MIB::icmpMsgStatsOutPkts.ipv6.3`)
rather than numeric. F1a offered "stored as text and sorts as text" as a
hypothesis. Measured in row F1b, it is wrong in both halves, and the truth is
worse — see below.

## Symbolic OIDs (row F1b, fixed)

A walk taken with plain `snmpwalk` rather than `snmpwalk -On` names its objects.
`parseWalkLine` accepted any text before the `=` as an OID key, so those names
became MIB keys verbatim. Two consequences, both measured:

- **They do not sort as text.** `parseOIDParts` builds its arc list with
  `strconv.Atoi` and _skips_ every arc that fails, so `SNMPv2-MIB::sysORDescr.3`
  and `IP-MIB::icmpMsgStatsOutPkts.ipv6.3` both reduce to `[3]` and
  `compareOIDs` returns **0** — they compare equal. The sorted list the GET-NEXT
  binary search walks is therefore not ordered, and a chain through it goes
  backwards or skips rows.
- **They never reach the wire at all.** gosnmp cannot marshal a non-numeric OID:
  `MarshalMsg` returns `unable to marshal OID: Invalid object identifier` and a
  zero-byte packet. One such varbind fails the _whole_ response, so the agent
  answers nothing and the scanner times out. This is a discovery-killing defect,
  not a cosmetic ordering one.

The fix makes "every OID the MIB holds is numeric" an invariant at the single
point both `ParseWalkFile` and `ParseWalkContent` pass through. `NormalizeWalkOID`
resolves what the SMI allows without a MIB compiler — an object in the fixed
known set, or an RFC 2578 registration anchor (`SNMPv2-SMI::enterprises`,
`SNMPv2-SMI::transmission`, the bare `iso.` form net-snmp prints with no MIBs
loaded), each followed by a numeric tail. Anything else is refused at parse:
an object from a MIB nothing ships, or a **symbolic table index** such as
`IP-MIB::icmpMsgStatsOutPkts.ipv6.3`, which no name table can resolve.

`validateOID` previously called a named OID "valid format, no action needed",
which is how these walks passed `niac sanitize --check` and the catalog-sync
gate. A resolvable name is now a `warning` carrying the numeric form as an
auto-fix; an unresolvable one is an `error` that names `snmpwalk -On`.

Measured over the four Nexus walks, before and after:

| Walk | Symbolic rows | Rejected after | Recovered | Ordering break |
| --- | --- | --- | --- | --- |
| `cisco-nexus-5000-05` | 34,990 | 10,028 | 24,962 | fixed |
| `cisco-nexus-7000-02` | 16,430 | 1,390 | 15,040 | fixed |
| `cisco-nexus-4000-02` | 3,765 | 519 | 3,246 | fixed |
| `cisco-nexus-7000-01` | 1,139 | 1,139 | 0 | fixed |

The harness gained a `rejected` count read from the raw file, because a
rejected row is absent from _both_ sides of the comparison: without it,
`cisco-nexus-7000-02` would report as byte-perfect having lost 1,390 rows.

**Left for the owner.** 240 distinct object names appear across those four
walks, from 7 standard MIB modules; the anchors above resolve 63% of the rows
and the fixed known set most of the rest, but standard objects outside it —
`SNMPv2-MIB::sysUpTime`, `IF-MIB::ifInOctets` — are still refused. Shipping a
~200-entry MIB-II name table would recover them and is new capability under the
plan's rule 8, so it is a decision, not a fix. The alternative, and the one the
error message states, is that a capture must be taken with `-On`.

A harness correction worth recording: the first corpus run reported 1.29
million dropped OIDs. Every one belonged to a walk whose sweep had hit a fixed
step cap, so the OIDs the sweep never reached looked dropped. The budget is now
derived from the walk's own size, and a sweep that still stops early is marked
`incomplete` with its verdict counts cleared rather than reported.

## What F0 does not do

No behaviour changes. The five predicates were lifted into the classifier and
their call sites replaced; `go test ./internal/protocols/snmp` is unchanged and
green. Measuring the contract is row F1a; fixing what it finds is F1b.
