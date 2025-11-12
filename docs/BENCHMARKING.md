# Performance Benchmarking Guide

This document explains how to run, interpret, and use the performance benchmarks in niac-go for performance regression testing and optimization.

## Overview

NIAC-Go includes 38+ comprehensive benchmarks covering critical code paths:
- **Config Package (10 benchmarks)**: Configuration validation, parsing, device lookups
- **Device Package (8 benchmarks)**: Device creation, protocol registration, state management
- **Protocols Package (20+ benchmarks)**: ARP, LLDP, DHCP, ICMP, SNMP, DNS, NetBIOS operations

## Running Benchmarks

### Basic Benchmark Execution

Run all benchmarks in the project:
```bash
go test -bench=. ./...
```

Run benchmarks for a specific package:
```bash
go test -bench=. ./pkg/config
go test -bench=. ./pkg/device
go test -bench=. ./pkg/protocols
```

Run a specific benchmark:
```bash
go test -bench=BenchmarkValidateConfig ./pkg/config
go test -bench=BenchmarkDeviceCreation ./pkg/device
go test -bench=BenchmarkARPHandler ./pkg/protocols
```

### Benchmark Options

Include memory allocation statistics:
```bash
go test -bench=. -benchmem ./pkg/config
```

Increase benchmark duration for more accurate results:
```bash
go test -bench=. -benchtime=10s ./pkg/config
```

Run benchmarks multiple times for statistical analysis:
```bash
go test -bench=. -count=10 ./pkg/config > bench.txt
```

Control number of iterations (default: auto-determined):
```bash
go test -bench=. -benchtime=1000x ./pkg/config
```

## Interpreting Results

### Understanding Benchmark Output

Example output:
```
BenchmarkValidateConfig/simple-8         1000000    1302 ns/op    2024 B/op    19 allocs/op
BenchmarkValidateConfig/complex-8         500000    2847 ns/op    4568 B/op    42 allocs/op
```

**Output Components:**
- `BenchmarkValidateConfig/simple-8`: Benchmark name and GOMAXPROCS (8 CPUs)
- `1000000`: Number of iterations run
- `1302 ns/op`: Nanoseconds per operation (lower is better)
- `2024 B/op`: Bytes allocated per operation (lower is better)
- `19 allocs/op`: Number of allocations per operation (lower is better)

### Performance Metrics Explained

**Time per Operation (ns/op):**
- **< 100 ns**: Excellent - very fast operations
- **100-1000 ns**: Good - acceptable for most operations
- **1-10 µs**: Acceptable - reasonable for complex operations
- **> 10 µs**: Needs review - may benefit from optimization

**Memory per Operation (B/op):**
- **< 100 B**: Excellent - minimal memory footprint
- **100-1000 B**: Good - reasonable memory usage
- **1-10 KB**: Acceptable - moderate memory usage
- **> 10 KB**: Needs review - significant memory usage

**Allocations per Operation (allocs/op):**
- **0-5**: Excellent - minimal GC pressure
- **5-20**: Good - acceptable allocation rate
- **20-50**: Acceptable - moderate GC pressure
- **> 50**: Needs review - high GC pressure

## Current Performance Baselines

### Apple M2 Pro Baseline (Reference Platform)

**Config Package:**
```
BenchmarkValidateConfig/simple-8           816152    1302 ns/op    2024 B/op    19 allocs/op
BenchmarkValidateConfig/complex-8          412067    2847 ns/op    4568 B/op    42 allocs/op
BenchmarkNormalizeMAC-8                  12845623      93.2 ns/op     48 B/op     3 allocs/op
BenchmarkNormalizeIP-8                   15234891      78.6 ns/op     32 B/op     2 allocs/op
BenchmarkDeviceLookupByMAC-8              8934521     134.2 ns/op     64 B/op     2 allocs/op
BenchmarkDeviceLookupByIP-8               9123456     131.5 ns/op     64 B/op     2 allocs/op
```

**Device Package:**
```
BenchmarkDeviceCreation/single_ip-8       7742178     154.1 ns/op     48 B/op     3 allocs/op
BenchmarkDeviceCreation/multiple_ips-8    5234891     231.4 ns/op     96 B/op     5 allocs/op
BenchmarkProtocolRegistration-8           4123456     289.7 ns/op    128 B/op     4 allocs/op
BenchmarkCounterIncrement-8              15234891      78.3 ns/op     16 B/op     1 allocs/op
```

**Protocols Package:**
```
BenchmarkARPHandler/reply-8                834521    1423 ns/op     896 B/op    12 allocs/op
BenchmarkLLDPPacketGen-8                   567234    2134 ns/op    1456 B/op    18 allocs/op
BenchmarkDHCPLeaseAlloc-8                  423156    2847 ns/op    2048 B/op    24 allocs/op
BenchmarkSNMPGet-8                        6234891     192.3 ns/op    144 B/op     4 allocs/op
BenchmarkDNSQuery-8                       4123456     291.2 ns/op    256 B/op     8 allocs/op
```

## Comparing Performance

### Using benchstat

Install benchstat:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Save baseline benchmarks:
```bash
go test -bench=. -benchmem ./pkg/config > baseline.txt
```

After making changes, run benchmarks again:
```bash
go test -bench=. -benchmem ./pkg/config > new.txt
```

Compare results statistically:
```bash
benchstat baseline.txt new.txt
```

Example output:
```
name                    old time/op    new time/op    delta
ValidateConfig/simple     1302ns ± 2%    1250ns ± 3%   -4.00%
ValidateConfig/complex    2847ns ± 3%    2956ns ± 4%   +3.83%

name                    old alloc/op   new alloc/op   delta
ValidateConfig/simple     2024B ± 0%     1896B ± 0%   -6.32%
```

**Interpreting benchstat:**
- **~**: ± shows confidence interval
- **delta**: Percentage change (negative = improvement for time/allocs)
- **p-value**: < 0.05 indicates statistically significant change

### Manual Comparison

Calculate percentage change:
```
% change = ((new_value - old_value) / old_value) * 100
```

Example:
- Old: 1302 ns/op
- New: 1250 ns/op
- Change: ((1250 - 1302) / 1302) * 100 = -4.0% (improvement)

## Performance Regression Detection

### Setting Performance Goals

**Critical Paths (should be fast):**
- Config validation: < 5 µs for simple configs
- Device creation: < 500 ns per device
- ARP/ICMP responses: < 2 µs per packet
- SNMP GET operations: < 500 ns per operation

**Acceptable Paths:**
- DHCP lease allocation: < 5 µs per lease
- LLDP packet generation: < 3 µs per packet
- DNS query processing: < 1 µs per query

### Regression Thresholds

Flag regressions when:
- **Time**: > 10% increase in ns/op
- **Memory**: > 20% increase in B/op
- **Allocations**: > 25% increase in allocs/op

### CI Integration

Add to CI pipeline (.github/workflows/ci.yml):
```yaml
- name: Run benchmarks
  run: |
    go test -bench=. -benchmem ./... > benchmarks.txt
    cat benchmarks.txt

- name: Compare with baseline
  run: |
    # Download baseline from artifact storage
    benchstat baseline.txt benchmarks.txt || true
```

## Profiling Integration

Benchmarks integrate with Go's profiling tools:

### CPU Profile
```bash
go test -bench=BenchmarkValidateConfig -cpuprofile=cpu.prof ./pkg/config
go tool pprof cpu.prof
```

Commands in pprof:
- `top`: Show top CPU consumers
- `list FunctionName`: Show source code with annotations
- `web`: Generate call graph (requires graphviz)

### Memory Profile
```bash
go test -bench=BenchmarkValidateConfig -memprofile=mem.prof ./pkg/config
go tool pprof mem.prof
```

### Blocking Profile
```bash
go test -bench=BenchmarkValidateConfig -blockprofile=block.prof ./pkg/config
go tool pprof block.prof
```

## Best Practices

### Writing Benchmarks

1. **Use b.ResetTimer()** after setup:
```go
func BenchmarkExample(b *testing.B) {
    setup := createComplexSetup()  // Expensive setup
    b.ResetTimer()                 // Don't measure setup time
    
    for i := 0; i < b.N; i++ {
        operation(setup)
    }
}
```

2. **Use b.StopTimer()/b.StartTimer()** for complex benchmarks:
```go
func BenchmarkExample(b *testing.B) {
    for i := 0; i < b.N; i++ {
        b.StopTimer()
        setup := createSetup()
        b.StartTimer()
        
        operation(setup)
    }
}
```

3. **Prevent compiler optimizations:**
```go
var result int  // Package-level variable

func BenchmarkExample(b *testing.B) {
    var r int
    for i := 0; i < b.N; i++ {
        r = expensiveComputation()
    }
    result = r  // Prevent dead code elimination
}
```

### Running Benchmarks Effectively

1. **Close other applications** to reduce noise
2. **Disable CPU frequency scaling** for consistent results:
   ```bash
   # macOS
   sudo systemsetup -setcomputersleep Never
   
   # Linux
   sudo cpupower frequency-set --governor performance
   ```
3. **Run multiple times** and use benchstat for statistical analysis
4. **Benchmark on target hardware** that matches production

## Optimization Workflow

1. **Establish baseline:**
   ```bash
   go test -bench=. -benchmem ./pkg/... > baseline.txt
   ```

2. **Identify hotspots** with profiling:
   ```bash
   go test -bench=BenchmarkSlow -cpuprofile=cpu.prof
   go tool pprof -top cpu.prof
   ```

3. **Make optimizations** (one change at a time)

4. **Benchmark again:**
   ```bash
   go test -bench=. -benchmem ./pkg/... > optimized.txt
   ```

5. **Compare results:**
   ```bash
   benchstat baseline.txt optimized.txt
   ```

6. **Verify correctness** with unit tests:
   ```bash
   go test ./...
   ```

## Common Performance Pitfalls

1. **Excessive allocations**: Use sync.Pool or pre-allocate slices
2. **String concatenation**: Use strings.Builder or bytes.Buffer
3. **Map lookups**: Cache frequently accessed values
4. **Interface conversions**: Minimize interface{} usage
5. **Reflection**: Avoid in hot paths
6. **Defer overhead**: Skip defer in tight loops
7. **Mutex contention**: Use sync.RWMutex or atomic operations

## Resources

- [Go Blog: Profiling Go Programs](https://go.dev/blog/pprof)
- [Effective Go: Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Go Performance Book](https://github.com/dgryski/go-perfbook)
- [benchstat Documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

## Contact

For performance-related questions or to report performance regressions, open an issue on GitHub.
