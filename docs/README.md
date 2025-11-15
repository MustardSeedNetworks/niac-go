# NIAC-Go Documentation

Comprehensive documentation for Network In A Can - Go Edition.

## Quick Start

- [Installation & Setup](../README.md#installation)
- [First Simulation](../README.md#quick-start)
- [Configuration Guide](../ARCHITECTURE.md#configuration)

## Core Documentation

### User Guides

- **[FAQ](FAQ.md)** - Frequently asked questions and troubleshooting
- **[API Examples](API-EXAMPLES.md)** - curl, Python, JavaScript, Go, PowerShell examples
- **[Performance Tuning](PERFORMANCE.md)** - Optimization guide for different workloads
- **[Deployment Guide](DEPLOYMENT.md)** - Docker, Kubernetes, systemd, cloud deployments
- **[CI/CD Integration](CI-CD.md)** - GitHub Actions, GitLab CI, Jenkins examples

### Protocol Guides

- **[SNMP Walk Files](SNMP-WALKS.md)** - Creating, optimizing, and contributing walk files
- **DNS Configuration** - See [examples/dns-server.yaml](../examples/)
- **DHCP Configuration** - See [examples/dhcp-server.yaml](../examples/)
- **HTTP/FTP Configuration** - See [examples/](../examples/)

### Development

- **[Architecture](../ARCHITECTURE.md)** - System design and internals
- **[Contributing](../CONTRIBUTING.md)** - How to contribute
- **[Breaking Changes Policy](BREAKING-CHANGES.md)** - Versioning and compatibility

## API Reference

### REST API

- **Base URL**: `http://localhost:8080/api/v1/`
- **Authentication**: Bearer token in `Authorization` header
- **CSRF Protection**: Required for POST/PUT/PATCH/DELETE

**Endpoints:**

| Method | Endpoint | Description | Auth | CSRF |
|--------|----------|-------------|------|------|
| GET | `/stats` | System statistics | Yes | No |
| GET | `/devices` | List devices | Yes | No |
| GET | `/config` | Get configuration | Yes | No |
| PUT | `/config` | Update configuration | Yes | Yes |
| GET | `/replay` | Replay status | Yes | No |
| POST | `/replay` | Start PCAP replay | Yes | Yes |
| DELETE | `/replay` | Stop PCAP replay | Yes | Yes |
| GET | `/csrf-token` | Get CSRF token | Yes | No |
| GET | `/version` | API version | Yes | No |

See [API-EXAMPLES.md](API-EXAMPLES.md) for detailed usage.

### Prometheus Metrics

- **Endpoint**: `http://localhost:9090/metrics`
- **Authentication**: None
- **Format**: Prometheus text format

**Available Metrics:**

- `niac_packets_sent_total` - Total packets sent
- `niac_packets_received_total` - Total packets received
- `niac_arp_requests_total` - ARP requests processed
- `niac_icmp_requests_total` - ICMP (ping) requests processed
- `niac_dns_queries_total` - DNS queries processed
- `niac_dhcp_requests_total` - DHCP requests processed
- `niac_errors_total` - Total errors encountered

## Configuration Reference

### Device Types

- `router` - Layer 3 routing device
- `switch` - Layer 2 switching device
- `server` - End host / server
- `firewall` - Security appliance
- `loadbalancer` - Load balancing device

### Protocol Configuration

Each device supports:
- **ARP**: Automatic (responds to ARP requests)
- **ICMP**: Automatic (responds to ping)
- **DNS**: Configure via `dns_config`
- **DHCP**: Configure via `dhcp_config`
- **HTTP**: Configure via `http_config`
- **FTP**: Configure via `ftp_config`
- **SNMP**: Configure via `snmp_config`
- **LLDP**: Configure via `lldp_config`
- **CDP**: Configure via `cdp_config`

See [examples/](../examples/) for complete configuration examples.

## Command-Line Reference

```bash
# Run simulation
niac run config.yaml [--api ADDR] [--debug LEVEL]

# Validate configuration
niac config validate config.yaml

# Configuration tools
niac config export config.yaml
niac config diff config1.yaml config2.yaml
niac config merge base.yaml overlay.yaml

# Version info
niac version

# Generate man pages
niac man
```

## Troubleshooting

### Common Issues

**Permission Denied**
```bash
# Solution: Run with sudo or grant capabilities
sudo niac run config.yaml
# OR
sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac
```

**Port Already in Use**
```bash
# Find process using port
sudo lsof -i :8080
# Kill or use different port
niac run config.yaml --api :8081
```

**High Memory Usage**
- Check goroutine count: `GET /api/v1/stats`
- Reduce device count or debug level
- See [PERFORMANCE.md](PERFORMANCE.md)

**Devices Not Responding**
- Verify interface is up: `ip link show`
- Check firewall rules: `sudo iptables -L`
- Enable debug: `--debug 2`

## Performance Benchmarks

| Metric | Value |
|--------|-------|
| Startup Time | ~5ms |
| Memory (idle) | ~15MB |
| Memory (100 devices) | ~50MB |
| Error Injection Rate | 7.7M/sec |
| Packet Processing | ~100K pps/core |
| ARP/ICMP Response | ~50K req/sec |

See [PERFORMANCE.md](PERFORMANCE.md) for tuning guide.

## Security

### Authentication

API requires Bearer token authentication:
```bash
export NIAC_API_TOKEN=$(openssl rand -base64 32)
niac run config.yaml --api :8080
```

### CSRF Protection

State-changing operations require CSRF token:
1. GET `/api/v1/csrf-token` to retrieve token
2. Include `X-CSRF-Token` header in POST/PUT/PATCH/DELETE requests

### Rate Limiting

- **Default**: 100 requests/second per IP
- **Burst**: 200 requests
- **Response**: HTTP 429 when exceeded

### Security Headers

All API responses include:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (with HTTPS)
- `Content-Security-Policy`

## Support & Community

- **Issues**: [GitHub Issues](https://github.com/krisarmstrong/niac-go/issues)
- **Discussions**: [GitHub Discussions](https://github.com/krisarmstrong/niac-go/discussions)
- **Changelog**: [CHANGELOG.md](../CHANGELOG.md)
- **License**: MIT License

## Version History

- **v2.6.0** - Quality improvements, documentation, monitoring enhancements
- **v2.5.0** - Defensive security improvements
- **v2.4.1** - Rate limiter security fixes
- **v2.4.0** - API rate limiting, error standardization
- **v2.3.2** - HIGH security fixes
- **v2.3.1** - CRITICAL security fixes

See [CHANGELOG.md](../CHANGELOG.md) for complete history.
