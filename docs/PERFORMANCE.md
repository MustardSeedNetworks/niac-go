# Performance Tuning Guide

This guide helps you optimize NIAC-Go for different deployment scenarios and traffic volumes.

## Quick Reference

| Traffic Level | Devices | Buffer Size | Memory | CPU Cores |
|--------------|---------|-------------|---------|-----------|
| Low          | 1-10    | 500         | 512MB   | 1         |
| Normal       | 10-50   | 1000        | 1-2GB   | 2         |
| High         | 50-200  | 5000        | 4-8GB   | 4         |
| Very High    | 200+    | 10000       | 8-16GB  | 8+        |

## Channel Buffer Sizing

Channel buffers control packet queue sizes. Adjust in `pkg/protocols/stack.go`:

```go
const DefaultQueueBufferSize = 1000  // Default for normal traffic
```

### Recommended Sizes

**Low Traffic (< 100 pps)**
```go
const DefaultQueueBufferSize = 500
```
- Use case: Lab testing, development
- Memory: ~512MB
- Latency: < 1ms

**Normal Traffic (100-1000 pps)**
```go
const DefaultQueueBufferSize = 1000  // Default
```
- Use case: Production simulation, testing
- Memory: ~1-2GB
- Latency: < 5ms

**High Traffic (1000-10000 pps)**
```go
const DefaultQueueBufferSize = 5000
```
- Use case: Load testing, stress testing
- Memory: ~4-8GB
- Latency: < 10ms

**Very High Traffic (> 10000 pps)**
```go
const DefaultQueueBufferSize = 10000
```
- Use case: Enterprise simulation, network chaos testing
- Memory: ~8-16GB
- Latency: < 20ms

## Debug Level Impact

Debug logging has significant performance impact:

| Level | Description | Overhead | Use Case |
|-------|-------------|----------|----------|
| 0     | No debug    | 0%       | Production |
| 1     | Basic       | 5-10%    | Monitoring |
| 2     | Detailed    | 15-25%   | Troubleshooting |
| 3     | Verbose     | 40-60%   | Development |

**Recommendation**: Use level 0 in production, level 1 for monitoring, level 2+ only for debugging.

## Protocol Optimization

### Disable Unused Protocols

Only enable protocols you need:

```yaml
devices:
  - name: router1
    type: router
    # Disable unused protocols
    dns_config: null
    ftp_config: null
    snmp_config: null  # If not needed
```

### SNMP Walk Files

Loading large walk files (> 10MB) can slow startup:

```yaml
snmp_config:
  community: public
  walk_file: /path/to/large.walk  # Pre-filter to essential OIDs
```

**Optimization**: Filter walk files to only essential OIDs:
```bash
grep -E '(sysDescr|ifDescr|ipAdEntAddr)' full.walk > filtered.walk
```

### DNS Record Limits

Limit DNS records per device to < 1000:

```yaml
dns_config:
  records:  # Keep under 1000 records per device
    - name: example.com
      type: A
      value: 192.168.1.1
```

## Memory Optimization

### Rate Limiter Cleanup

Rate limiter automatically cleans up stale entries every 5 minutes. For high-traffic scenarios, entries not seen in 1 hour are removed.

Monitor with:
```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats | jq .goroutines
```

### Goroutine Limits

Healthy goroutine counts:
- **< 100**: Normal operation
- **100-500**: High traffic, monitor
- **> 500**: Investigate potential leaks

Check goroutines via API:
```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats | jq .goroutines
```

## CPU Optimization

### GOMAXPROCS

Set CPU core usage:
```bash
# Use all cores (default)
sudo niac run config.yaml

# Limit to 4 cores
GOMAXPROCS=4 sudo niac run config.yaml
```

### Process Priority

Linux - increase priority:
```bash
sudo nice -n -10 niac run config.yaml
```

### CPU Affinity

Pin to specific cores:
```bash
taskset -c 0-3 sudo niac run config.yaml
```

## Storage Optimization

### Use SSD for Storage Backend

```bash
sudo niac run config.yaml --storage /path/to/ssd/storage.db
```

### Limit Run History

```yaml
# In future config option
storage:
  max_runs: 100  # Keep last 100 runs only
```

## Network Interface Optimization

### Interface Ring Buffers

Increase RX ring buffer size:
```bash
# Check current size
ethtool -g eth0

# Increase to maximum
sudo ethtool -G eth0 rx 4096 tx 4096
```

### Disable TCP Offloading

For packet capture accuracy:
```bash
sudo ethtool -K eth0 rx off tx off sg off tso off gso off gro off
```

## Monitoring Performance

### Built-in Metrics

Prometheus metrics endpoint:
```bash
curl http://localhost:9090/metrics
```

Key metrics:
- `niac_packets_sent_total` - Total packets sent
- `niac_packets_received_total` - Total packets received
- `niac_errors_total` - Total errors
- `niac_goroutines` - Current goroutine count

### Runtime Profiling

Enable pprof:
```bash
sudo niac run config.yaml --pprof :6060
```

CPU profile:
```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

Memory profile:
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

Goroutine profile:
```bash
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Benchmarking

### Error Injection Performance

```bash
niac test error-rate --duration 10s --devices 10
# Expected: 5-10M errors/sec
```

### Packet Processing

```bash
# Generate test traffic
sudo tcpreplay -i eth0 -M 10 test.pcap

# Monitor processing rate
watch -n 1 'curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats | jq .stack.packets_received'
```

## Tuning by Use Case

### Lab/Development
```yaml
# Minimal resource usage
debug: 2
devices: 1-5
protocols: only needed ones
```

### CI/CD Testing
```yaml
# Fast startup, deterministic
debug: 0
devices: < 10
storage: memory  # No disk I/O
```

### Load Testing
```yaml
# High throughput
debug: 0
devices: 50-200
channel_buffers: 5000
```

### Production Simulation
```yaml
# Balanced performance/observability
debug: 1
devices: 50-100
channel_buffers: 1000
metrics: enabled
```

## Troubleshooting Performance Issues

### High CPU Usage

1. Check debug level: `--debug 0`
2. Reduce device count
3. Disable unused protocols
4. Check for broadcast storms

### High Memory Usage

1. Monitor goroutines: `GET /api/v1/stats`
2. Check rate limiter size (logs show cleanup activity)
3. Reduce SNMP walk file sizes
4. Limit DNS records

### Packet Loss

1. Increase channel buffer size
2. Add more CPU cores
3. Check interface ring buffers
4. Monitor system load

### Slow API Responses

1. Check rate limiting (429 errors)
2. Reduce debug logging
3. Use connection pooling
4. Enable HTTP/2

## Best Practices

1. **Start small**: Begin with minimal config, scale up
2. **Monitor first**: Enable metrics before tuning
3. **One change at a time**: Measure impact of each change
4. **Document baselines**: Record initial performance
5. **Test under load**: Use realistic traffic patterns
6. **Profile regularly**: Check for resource leaks
7. **Plan capacity**: Size for 2-3x expected load
8. **Automate monitoring**: Alert on anomalies

## Performance Checklist

- [ ] Debug level set appropriately (0 for production)
- [ ] Channel buffers sized for traffic volume
- [ ] Unused protocols disabled
- [ ] SNMP walk files filtered/compressed
- [ ] Sufficient CPU cores allocated
- [ ] Memory headroom available (2-3x working set)
- [ ] Network interface optimized
- [ ] Monitoring/metrics enabled
- [ ] Baseline performance recorded
- [ ] Load testing completed
