# Benchmarking

NIAC keeps focused Go benchmarks beside the code they measure. Benchmark
results are meaningful only when the command, version, host, and scenario are
recorded together.

## Run the benchmark suites

```bash
go test -run '^$' -bench . -benchmem ./internal/config
go test -run '^$' -bench . -benchmem ./internal/capture
go test -run '^$' -bench . -benchmem ./internal/protocols
go test -run '^$' -bench . -benchmem ./internal/protocols/snmp
```

Run a focused benchmark by its current name:

```bash
go test -run '^$' -bench '^BenchmarkLoadYAML_Large$' -benchmem ./internal/config
go test -run '^$' -bench '^BenchmarkARPRequestHandling$' -benchmem ./internal/protocols
go test -run '^$' -bench '^BenchmarkAgent_HandleGetBulk$' -benchmem ./internal/protocols/snmp
```

Use `go test -list '^Benchmark' <package>` to discover the benchmarks in the
checked-out version instead of copying an old name from a report.

## Compare a change

Collect multiple samples from the same host:

```bash
go test -run '^$' -bench . -benchmem -count 10 ./internal/config > baseline.txt
go test -run '^$' -bench . -benchmem -count 10 ./internal/config > candidate.txt
benchstat baseline.txt candidate.txt
```

Do not compare results collected with different power modes, background load,
Go versions, architectures, or capture drivers without identifying that
difference.

## Profile a focused benchmark

```bash
go test -run '^$' \
  -bench '^BenchmarkLoadYAML_Large$' \
  -cpuprofile cpu.prof \
  -memprofile mem.prof \
  ./internal/config

go tool pprof cpu.prof
go tool pprof mem.prof
```

Block and mutex profiles are useful only after a concurrency symptom has been
reproduced:

```bash
go test -run '^$' \
  -bench '^BenchmarkARPRequestHandling$' \
  -blockprofile block.prof \
  -mutexprofile mutex.prof \
  ./internal/protocols
```

## Writing benchmarks

- Benchmark a real public or package-level behavior.
- Keep setup outside the timed region and use `b.ResetTimer` when necessary.
- Report allocations with `b.ReportAllocs`.
- Consume results so the compiler cannot eliminate the measured work.
- Use representative small and large inputs.
- Avoid network, disk, sleeps, and logging unless that I/O is the behavior
  being measured.
- Pair performance work with correctness and race tests.

## Reporting results

Include:

- `go version`;
- NIAC commit and whether the tree was dirty;
- OS, architecture, CPU, memory, and power mode;
- exact benchmark command;
- `benchstat` output or raw samples;
- the acceptance threshold and why it matters to an operator workflow.

Repository benchmark output is engineering evidence, not a general packet-rate
or device-capacity claim. End-to-end capacity must be measured in the target
lab with the observer and attachment described in [PERFORMANCE.md](PERFORMANCE.md).
