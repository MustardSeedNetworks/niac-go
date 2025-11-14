# NIAC-Go Monitoring Guide

This guide explains how to monitor NIAC-Go using Prometheus and Grafana.

## Table of Contents

- [Overview](#overview)
- [Prometheus Metrics](#prometheus-metrics)
- [Prometheus Setup](#prometheus-setup)
- [Grafana Setup](#grafana-setup)
- [Available Metrics](#available-metrics)
- [Alerting](#alerting)

## Overview

NIAC-Go exposes comprehensive metrics via a Prometheus-compatible `/metrics` endpoint. These metrics provide visibility into:

- **Traffic Statistics**: Packet counts, protocol breakdown
- **System Performance**: Memory usage, goroutines, GC activity
- **Runtime Metrics**: Uptime, error counts
- **Protocol Details**: ARP, ICMP, DNS, DHCP, SNMP activity

## Prometheus Metrics

The metrics endpoint is available at `http://localhost:8080/metrics` (or your configured API port).

### Quick Test

```bash
# View raw metrics
curl http://localhost:8080/metrics

# Example output:
# HELP niac_packets_sent_total Total packets sent
# TYPE niac_packets_sent_total counter
niac_packets_sent_total 15234
# HELP niac_memory_usage_bytes Memory usage in bytes
# TYPE niac_memory_usage_bytes gauge
niac_memory_usage_bytes 45678912
...
```

## Prometheus Setup

### 1. Install Prometheus

**macOS (Homebrew):**
```bash
brew install prometheus
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install prometheus
```

**Docker:**
```bash
docker run -d -p 9090:9090 \
  -v /path/to/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

### 2. Configure Prometheus

Create or edit `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'niac-go'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s
```

**Important**: Make sure the `targets` address matches your NIAC-Go API server address.

### 3. Start Prometheus

```bash
# Using local installation
prometheus --config.file=prometheus.yml

# Using Docker
docker run -d -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  --name prometheus prom/prometheus
```

### 4. Verify Scraping

1. Open Prometheus UI: http://localhost:9090
2. Go to **Status** → **Targets**
3. Verify `niac-go` target shows **UP**
4. Query a metric like `niac_devices_total` to test

## Grafana Setup

### 1. Install Grafana

**macOS (Homebrew):**
```bash
brew install grafana
brew services start grafana
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install grafana
sudo systemctl start grafana-server
sudo systemctl enable grafana-server
```

**Docker:**
```bash
docker run -d -p 3000:3000 --name=grafana grafana/grafana
```

### 2. Access Grafana

1. Open Grafana: http://localhost:3000
2. Default credentials: `admin` / `admin` (change on first login)

### 3. Add Prometheus Data Source

1. Navigate to **Configuration** → **Data Sources**
2. Click **Add data source**
3. Select **Prometheus**
4. Configure:
   - **Name**: `Prometheus`
   - **URL**: `http://localhost:9090`
   - Click **Save & Test**

### 4. Import NIAC-Go Dashboard

Option A: Import from file
1. Download `docs/grafana-dashboard.json` from the repository
2. Navigate to **Create** → **Import**
3. Click **Upload JSON file**
4. Select the dashboard file
5. Select your Prometheus data source
6. Click **Import**

Option B: Manual import
1. Navigate to **Create** → **Import**
2. Paste the contents of `grafana-dashboard.json`
3. Select your Prometheus data source
4. Click **Import**

### 5. View Dashboard

The dashboard will display:
- **Overview Panel**: Devices, packet rates, errors
- **System Metrics Panel**: Goroutines, memory, uptime, GC
- **Protocol Breakdown Panel**: Traffic by protocol type
- **Memory Usage Panel**: Memory trends
- **Runtime Metrics Panel**: Goroutines and GC over time

## Available Metrics

### Traffic Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `niac_packets_sent_total` | counter | Total packets sent |
| `niac_packets_received_total` | counter | Total packets received |
| `niac_devices_total` | gauge | Number of simulated devices |
| `niac_errors_total` | counter | Total errors |

### Protocol-Specific Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `niac_arp_requests_total` | counter | ARP requests sent |
| `niac_arp_replies_total` | counter | ARP replies sent |
| `niac_icmp_requests_total` | counter | ICMP requests sent |
| `niac_icmp_replies_total` | counter | ICMP replies sent |
| `niac_dns_queries_total` | counter | DNS queries processed |
| `niac_dhcp_requests_total` | counter | DHCP requests processed |
| `niac_snmp_queries_total` | counter | SNMP queries processed |

### System Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `niac_uptime_seconds` | gauge | Server uptime in seconds |
| `niac_goroutines_total` | gauge | Number of active goroutines |
| `niac_memory_usage_bytes` | gauge | Current memory allocation |
| `niac_memory_sys_bytes` | gauge | Total memory from OS |
| `niac_gc_runs_total` | counter | Total garbage collection runs |

## Alerting

### Prometheus Alert Rules

Create `alert-rules.yml`:

```yaml
groups:
  - name: niac_alerts
    interval: 30s
    rules:
      # High memory usage
      - alert: HighMemoryUsage
        expr: niac_memory_usage_bytes / niac_memory_sys_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "NIAC-Go high memory usage"
          description: "Memory usage is above 90% for 5 minutes"

      # High goroutine count
      - alert: HighGoroutineCount
        expr: niac_goroutines_total > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "NIAC-Go high goroutine count"
          description: "Goroutine count exceeds 1000"

      # Error rate increase
      - alert: HighErrorRate
        expr: rate(niac_errors_total[5m]) > 10
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "NIAC-Go high error rate"
          description: "Error rate exceeds 10/sec"

      # Device simulation down
      - alert: NoDevices
        expr: niac_devices_total == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "NIAC-Go has no devices"
          description: "No devices are being simulated"
```

Add to `prometheus.yml`:

```yaml
rule_files:
  - "alert-rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']  # If using Alertmanager
```

### Webhook Alerts (Built-in)

NIAC-Go has a built-in webhook alerting system for packet thresholds.

Configure via REST API:

```bash
# Set alert threshold
curl -X PUT http://localhost:8080/api/v1/alert \
  -H "Content-Type: application/json" \
  -d '{
    "packets_threshold": 100000,
    "webhook_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  }'

# Get alert status
curl http://localhost:8080/api/v1/alert

# Disable alerts
curl -X DELETE http://localhost:8080/api/v1/alert
```

## Useful Queries

### Packet Rate
```promql
rate(niac_packets_sent_total[5m])
rate(niac_packets_received_total[5m])
```

### Protocol Distribution
```promql
# ARP percentage
(rate(niac_arp_requests_total[5m]) + rate(niac_arp_replies_total[5m]))
/ rate(niac_packets_sent_total[5m]) * 100
```

### Memory Growth
```promql
# Memory growth over 1 hour
delta(niac_memory_usage_bytes[1h])
```

### GC Frequency
```promql
# GC runs per minute
rate(niac_gc_runs_total[5m]) * 60
```

### Uptime
```promql
# Uptime in hours
niac_uptime_seconds / 3600
```

## Troubleshooting

### Metrics endpoint returns 503

The simulation is not running. Start a simulation:

```bash
niac --config devices.yml
```

### Prometheus shows target down

1. Check NIAC-Go is running: `curl http://localhost:8080/api/v1/status`
2. Verify API port in `prometheus.yml` matches your configuration
3. Check firewall rules

### No data in Grafana

1. Verify Prometheus data source is configured correctly
2. Check Prometheus is scraping: http://localhost:9090/targets
3. Verify metrics exist: Query `niac_devices_total` in Prometheus

### High memory usage

This is normal during large simulations. Consider:
- Reducing number of simulated devices
- Disabling packet capture
- Increasing available memory

## Best Practices

1. **Scrape Interval**: Use 5-15s for real-time monitoring
2. **Retention**: Configure Prometheus retention based on your needs:
   ```bash
   prometheus --storage.tsdb.retention.time=30d
   ```
3. **High Availability**: Run multiple Prometheus instances for production
4. **Grafana Alerts**: Set up Grafana alerting in addition to Prometheus
5. **Metric Cardinality**: NIAC-Go metrics have low cardinality, safe for long-term storage

## Related Documentation

- [REST API Documentation](REST_API.md)
- [Configuration Guide](../README.md#configuration)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

## Support

For monitoring-related issues, please open an issue on GitHub with:
- Prometheus/Grafana versions
- Configuration files (sanitize sensitive data)
- Relevant log output
