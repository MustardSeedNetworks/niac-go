# Recording corpus replay

`niac-corpus-replay` replays SNMP recordings through NIAC's walk readers and
SNMP agent and reports, per recording, whether every recorded binding reaches
the wire intact. Its main use is the qualified recording corpus in
[`niac-demo-catalog`](SHARED_DEMO_CATALOG.md): every recording there qualified
with a replay receipt, and this command checks that the current reader still
reproduces each one.

The recordings come from several sources under several licences, so they stay
in the catalog. Nothing here ships or copies them.

## What one replay checks

For each recording the command:

1. Loads it into a fresh agent with `LoadWalkFile`, the same path a scenario
   uses. net-snmp walks and snmpsim `.snmprec` files are both accepted; a
   `.snmprec.bz2` is expanded to a private temporary file first.
2. Reads the recording a second time with an independent record reader and
   asks the agent for every binding with `GetRequest`, 64 at a time.
3. BER-encodes each response and decodes it again, counting bindings that
   cannot be encoded (`ber_failures`) or that change type or value on the way.
4. For OIDs the [walk contract](SNMP_WALKS.md) keeps, requires the recorded
   type and value back (`kept_type_changes`, `kept_value_changes`). OIDs the
   contract substitutes are counted separately (`contract_substituted`).
5. Walks the agent with `GetNext` from the root and `GetBulk` from four roots,
   counting any binding that does not advance (`ordering_failures`).
6. With `-smoke`, serves the agent on an ephemeral loopback UDP port and sends
   a Get, GetNext and GetBulk with a real client (`udp_smoke`).

A panic anywhere in the reader or agent is caught and reported as status
`panic` rather than stopping the run.

The expected value of a net-snmp STRING is taken as the literal bytes between
its quotes, not through the production string decoder, so a decoder bug that
changes both sides equally still shows as a kept-value change.

## Replaying the catalog

Check out the catalog next to this repository (or anywhere) and point the
command at it:

```sh
git clone git@github.com:MustardSeedNetworks/niac-demo-catalog.git ../niac-demo-catalog
go run ./cmd/niac-corpus-replay -catalog ../niac-demo-catalog -smoke \
  -out build/corpus-replay/results.jsonl -summary build/corpus-replay/summary.json
```

`-catalog` defaults to `$NIAC_DEMO_CATALOG_DIR`. The command reads
`runtime/niac/qualified.csv` and replays each row's `effective_path` from
`walks/`. A repaired recording replays from `walks/repaired/` but is reported
under its `original_path`.

`make corpus-replay` runs the same thing with `-skip-missing`, using
`NIAC_DEMO_CATALOG_DIR` or `.catalog/niac-demo-catalog`, and writes to
`build/corpus-replay/`.

Useful flags:

| Flag | Effect |
| --- | --- |
| `-match TEXT` | Replay only recordings whose path contains `TEXT`. |
| `-limit N` | Replay at most `N` recordings after `-match`. |
| `-jobs N` | Replay `N` recordings concurrently (default `GOMAXPROCS`). |
| `-smoke` | Add the loopback UDP client test. |
| `-timing` | Include per-recording wall time. |
| `-skip-missing` | Exit 0 without replaying when the catalog is absent. |
| `-v` | Keep agent logging (discarded by default). |

Recordings can also be named directly, with no manifest and so no baseline:

```sh
go run ./cmd/niac-corpus-replay path/to/device.snmprec path/to/other.walk
```

## Output

Standard output (or `-out`) holds one JSON object per recording, sorted by
path:

```json
{"path":"external/librenms/3com.snmprec","effective_path":"external/librenms/3com.snmprec","receipt":{"status":"completed","source_entries":12,"rejected_lines":0,"invalid_oids":0,"probe_bindings":12,"get_missing":0,"contract_omitted":0,"opaque_wire_preserved":0,"ber_failures":0,"ber_type_changes":0,"ber_value_changes":0,"contract_kept":12,"contract_substituted":0,"kept_type_changes":0,"kept_value_changes":0,"source_identity":true,"next_checks":16,"bulk_checks":101,"ordering_failures":0,"udp_smoke":true,"failure_types":{}}}
```

The receipt fields have the same names as the qualification columns in
`qualified.csv`. Each result can also carry:

- `source_changed`: the file's SHA-256 no longer matches the `source_sha256`
  it qualified with.
- `drift`: every baseline column the replay disagrees with, as
  `"field: baseline X, replay Y"`. Wall time, the qualifying platform and
  `net_snmp_client` are not compared; `udp_smoke` is compared only with
  `-smoke`.
- `error`: why a recording could not be loaded or read.

Output is deterministic: results are sorted, map keys are sorted, wall time is
left out unless `-timing` is given, and nothing depends on `-jobs`. Two runs
over the same catalog and reader diff cleanly.

A summary goes to standard error, and to `-summary` as JSON.

## Exit status

| Code | Meaning |
| --- | --- |
| 0 | Every recording completed with no drift, or the catalog was absent with `-skip-missing`. |
| 1 | A recording failed or drifted from its baseline. A changed source alone does not fail the run; its replay is still compared. |
| 2 | Bad flags, or the catalog or manifest could not be read. |

## In `go test`

`internal/corpusreplay` has a test that replays a whole catalog and fails on
any drift. It runs only when `NIAC_CORPUS_REPLAY_CATALOG` names a catalog
checkout and `-short` is not set, so CI and an ordinary `go test ./...` skip
it:

```sh
NIAC_CORPUS_REPLAY_CATALOG=../niac-demo-catalog go test ./internal/corpusreplay -run TestQualifiedCatalog -v
```

It uses its own variable rather than `NIAC_DEMO_CATALOG_DIR` so that keeping
the latter set for example syncing does not add a full corpus replay to every
test run.
