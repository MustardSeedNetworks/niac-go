# Performance Guide

NIAC performance depends on the active protocols, packet rate, SNMP walk size,
replay pacing, observer polling, and host/interface behavior. Use measurements
from the target lab instead of fixed capacity claims.

## Start with supported defaults

- Use debug level 0 for sustained runs; enable higher levels only while
  diagnosing a specific protocol.
- Enable only the responders required by the scenario.
- Keep SNMP walks sanitized and limited to the tables the observer needs.
- Use a replay rate mode and limit appropriate for the attachment.
- Keep browser filters narrow when inspecting large packet or device sets.
- Do not modify internal queue constants as an operator tuning mechanism.

NIAC Free supports ten simulated devices. NIAC Pro removes that tier soft cap,
but the absolute ceiling remains 1,000 devices. This is an authorization and
safety contract, not a promise that every 1,000-device scenario will meet the
same latency target on every host.

## Measure the running daemon

The daemon exposes Prometheus metrics on its HTTPS listener:

```bash
curl -sk \
  -H "Authorization: Bearer $NIAC_API_TOKEN" \
  https://localhost:8445/metrics
```

Runtime statistics are also available through `/api/v1/stats`. Track packet
counts, drops, active goroutines, replay progress, and observer-visible
response latency while reproducing the actual scenario.

For a release or comparison, record:

- NIAC version and commit from `/__version`;
- operating system, architecture, CPU, and memory;
- interface, link speed, offload settings, and capture driver;
- configuration/device count and enabled protocols;
- traffic generator and offered rate;
- observer and polling interval;
- exact command, duration, and raw result.

## Replay

Replay streams a capture instead of loading the whole file into memory. Rate
modes include original timing, top speed, packets per second, and Mbps caps.
Loop count and BPF filtering are applied by the same playback engine used by
the daemon.

During the first streaming pass, a total is not fabricated before the reader
has observed the file. Later passes can use the prior complete pass as their
progress total. Filtered packets are reported separately. Malformed or
truncated trailing records are excluded from totals while the valid prefix is
replayed.

## SNMP walks

Large walk files increase parsing time and MIB memory. Prefer the smallest
sanitized walk that preserves the identity and tables needed by the monitoring
workflow. Measure both startup time and steady-state poll latency; a fast start
does not prove a large GET-BULK workload is acceptable.

## Host networking

Packet capture accuracy is host- and driver-dependent. When diagnosing missing
or coalesced frames, record NIC offload settings and compare the host capture
with an external observer. Any offload change affects the host and should be
made deliberately outside NIAC, then restored after the test.

On Linux, CPU affinity or process priority can be useful for a controlled
benchmark, but they are deployment choices rather than NIAC configuration
fields. Record them with the result.

## Profiling

The legacy foreground CLI can expose Go pprof on loopback when its explicit
profiling flag is enabled. Never bind profiling to an untrusted network.
Daemon performance investigations should begin with metrics, traces, race
tests, and focused Go benchmarks; add profiling only for a reproduced problem.

## Regression workflow

1. Capture a baseline on an unchanged host.
2. Run the same benchmark command multiple times.
3. Apply one change.
4. Repeat with the same configuration and environment.
5. Compare distributions with `benchstat`.
6. Reject regressions that exceed the acceptance threshold for the affected
   workflow.

Repository benchmark commands are documented in
[BENCHMARKING.md](BENCHMARKING.md). Release acceptance uses the full lint,
race, browser, security, package, install, and deployment gates in addition to
performance measurements.
