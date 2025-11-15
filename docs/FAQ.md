# Frequently Asked Questions (FAQ)

## General Questions

### What is NIAC-Go?

NIAC-Go (Network In A Can - Go Edition) is a production-ready network device simulator that allows you to simulate routers, switches, servers, and other network devices for testing, development, and education purposes. It supports multiple protocols including ARP, ICMP, DNS, DHCP, HTTP, FTP, SNMP, LLDP, and CDP.

### Why was NIAC rewritten in Go?

The original NIAC was written in Java. The Go rewrite provides:
- **10x faster startup** (~5ms vs ~50ms)
- **6.7x less memory** (~15MB vs ~100MB)
- **77x faster error injection** (7.7M/sec vs ~100K/sec)
- **Single binary deployment** (no JRE required)
- Better concurrency with goroutines
- Modern terminal UI with Bubbletea

### What protocols are supported?

- **Layer 2**: ARP, LLDP, CDP
- **Layer 3**: ICMP (ping), IPv4, IPv6 (partial)
- **Network Services**: DNS, DHCP (IPv4)
- **Application**: HTTP, FTP, SNMP (v2c)
- **Monitoring**: SNMP traps, statistics

### What are the system requirements?

- **OS**: Linux, macOS, Windows
- **Go Version**: 1.25.4 or later (for building from source)
- **Memory**: ~15-50MB depending on configuration
- **Privileges**: Root/admin for packet capture (raw sockets)

## Installation & Setup

### How do I install NIAC-Go?

**Option 1: Download binary from GitHub releases**
```bash
# Download latest release for your platform
curl -L https://github.com/krisarmstrong/niac-go/releases/latest/download/niac-$(uname -s)-$(uname -m).tar.gz | tar xz
sudo mv niac /usr/local/bin/
```

**Option 2: Build from source**
```bash
git clone https://github.com/krisarmstrong/niac-go.git
cd niac-go
go build -o niac ./cmd/niac
sudo mv niac /usr/local/bin/
```

### Why do I need root/admin privileges?

NIAC-Go uses raw sockets to send and receive packets at layer 2, which requires elevated privileges on most operating systems. You can run it with:
- Linux: `sudo niac ...`
- macOS: `sudo niac ...`
- Windows: Run as Administrator

### Can I run NIAC-Go without root?

On Linux, you can grant specific capabilities to the binary:
```bash
sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac
```

This allows packet capture without full root privileges.

## Configuration

### How do I create a configuration file?

Use the interactive configuration wizard:
```bash
niac config wizard
```

Or start with an example:
```bash
# Copy example config
cp examples/basic-router.yaml my-config.yaml

# Edit with your favorite editor
vim my-config.yaml
```

### What's the difference between YAML and JSON configs?

NIAC-Go primarily uses YAML for human-readable configuration. JSON is supported for API interactions but YAML is recommended for file-based configs due to better readability and comment support.

### How do I validate my configuration?

```bash
niac config validate my-config.yaml
```

This checks for:
- YAML syntax errors
- Invalid IP addresses or MAC addresses
- Missing required fields
- Protocol-specific configuration issues

### Can I use environment variables in configs?

Yes! Use environment variable substitution:
```yaml
devices:
  - name: router-${ENVIRONMENT}
    ip_addresses:
      - ${ROUTER_IP}
```

Then run:
```bash
ENVIRONMENT=prod ROUTER_IP=192.168.1.1 niac run my-config.yaml
```

## Running & Operations

### How do I start a simulation?

```bash
# Basic usage
sudo niac run config.yaml

# With debug output
sudo niac run config.yaml --debug 2

# With API server
sudo niac run config.yaml --api :8080 --api-token $(openssl rand -base64 32)
```

### How do I stop a running simulation?

Press `Ctrl+C` or send SIGTERM:
```bash
sudo kill -TERM $(pgrep niac)
```

NIAC-Go handles graceful shutdown, closing network interfaces and saving state.

### Can I run multiple NIAC instances?

Yes, but each must use a different network interface or run in different network namespaces. Example:
```bash
# Instance 1 on eth0
sudo niac run config1.yaml --interface eth0

# Instance 2 on eth1
sudo niac run config2.yaml --interface eth1
```

### How do I monitor statistics?

**TUI (Terminal UI):**
```bash
sudo niac run config.yaml
# Press 's' for statistics view
```

**API:**
```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats
```

**Prometheus Metrics:**
```bash
curl http://localhost:9090/metrics
```

## Protocols

### How do I configure DNS responses?

```yaml
dns_config:
  enabled: true
  records:
    - name: example.com
      type: A
      value: 192.168.1.100
    - name: mail.example.com
      type: MX
      value: 10 mail.example.com
```

### How do I set up DHCP?

```yaml
dhcp_config:
  enabled: true
  pool_start: 192.168.1.100
  pool_end: 192.168.1.200
  subnet_mask: 255.255.255.0
  router: 192.168.1.1
  dns_servers:
    - 8.8.8.8
    - 8.8.4.4
  lease_time: 86400  # 24 hours
```

### How do I load SNMP walk files?

```yaml
snmp_config:
  community: public
  walk_file: /path/to/device.walk
```

Walk files contain OID-value pairs. Create them with:
```bash
snmpwalk -v2c -c public real-device .1 > device.walk
```

### Can I simulate SNMP traps?

Yes! Configure trap generation:
```yaml
trap_config:
  enabled: true
  receivers:
    - 192.168.1.50:162
  cold_start:
    enabled: true
    on_startup: true
  link_state:
    enabled: true
    link_down: true
    link_up: true
```

## API & Integration

### How do I authenticate with the API?

Set an API token:
```bash
export NIAC_API_TOKEN=$(openssl rand -base64 32)
sudo niac run config.yaml --api :8080
```

Then include it in requests:
```bash
curl -H "Authorization: Bearer $NIAC_API_TOKEN" http://localhost:8080/api/v1/stats
```

### What API endpoints are available?

- `GET /api/v1/stats` - Statistics
- `GET /api/v1/devices` - Device list
- `GET /api/v1/config` - Current config
- `PUT /api/v1/config` - Update config
- `GET /api/v1/replay` - Replay status
- `POST /api/v1/replay` - Start replay
- `DELETE /api/v1/replay` - Stop replay
- `GET /api/v1/csrf-token` - Get CSRF token

See `docs/API.md` for full documentation.

### Is there rate limiting?

Yes. The API enforces:
- **100 requests/second per IP** (default)
- **Burst of 200 requests**

Exceeding limits returns HTTP 429 (Too Many Requests).

### How do I integrate with monitoring systems?

**Prometheus:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'niac'
    static_configs:
      - targets: ['localhost:9090']
```

**Custom webhooks:**
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"webhook_url":"http://alertmanager:9093/api/v1/alerts"}' \
  http://localhost:8080/api/v1/alerts
```

## Troubleshooting

### Why am I getting "permission denied" errors?

You need root/admin privileges for raw socket access:
```bash
sudo niac run config.yaml
```

Or grant capabilities (Linux only):
```bash
sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac
```

### Why aren't my devices responding?

Check:
1. **Interface is up**: `ip link show eth0`
2. **Firewall rules**: `sudo iptables -L`
3. **IP addresses don't conflict**: Use different subnets
4. **Debug output**: `sudo niac run config.yaml --debug 2`

### How do I enable debug logging?

Use the `--debug` flag with level 1-3:
```bash
# Level 1: Basic info
sudo niac run config.yaml --debug 1

# Level 2: Detailed (recommended)
sudo niac run config.yaml --debug 2

# Level 3: Verbose (very detailed)
sudo niac run config.yaml --debug 3
```

Per-protocol debugging:
```yaml
debug:
  dns: 3
  dhcp: 2
  http: 1
```

### Why is my PCAP replay not working?

Check:
1. **PCAP file is valid**: `tcpdump -r file.pcap | head`
2. **File size under 100MB**: Larger files are rejected
3. **Interface matches**: Ensure interface in PCAP exists
4. **Replay engine available**: Not available in daemon mode

### Memory usage keeps growing - is this a leak?

Likely not. Check:
- **Goroutine count**: `GET /api/v1/stats` shows goroutine count
- **Rate limiter**: Automatically cleaned up every 5 minutes
- **Statistics**: Growing counters are expected

Enable goroutine profiling:
```bash
sudo niac run config.yaml --pprof :6060
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### How do I report a bug?

1. Check existing issues: https://github.com/krisarmstrong/niac-go/issues
2. Gather information:
   - NIAC-Go version: `niac version`
   - OS and version: `uname -a`
   - Configuration file (sanitized)
   - Debug logs: `--debug 3`
3. Create new issue with details

## Performance

### How many devices can I simulate?

Depends on resources, but typical limits:
- **Small**: 10-50 devices on 2GB RAM
- **Medium**: 50-200 devices on 4GB RAM
- **Large**: 200-1000 devices on 8GB+ RAM

### How do I optimize performance?

See `docs/PERFORMANCE.md` for detailed tuning guide. Quick tips:
- Increase channel buffer sizes for high traffic
- Use higher debug levels only when needed
- Disable unused protocols
- Use SSD for storage backend

### What's the packet processing rate?

- **Error injection**: 7.7M errors/second
- **Packet processing**: ~100K packets/second (single core)
- **ARP/ICMP**: ~50K requests/second

## WebUI

### How do I access the WebUI?

Start API server, then visit in browser:
```bash
sudo niac run config.yaml --api :8080 --api-token $TOKEN
# Visit: http://localhost:8080
```

### Why is the WebUI slow with many devices?

For 100+ devices, performance may degrade. Upcoming improvements:
- Virtual scrolling (#126)
- React re-render optimization (#125)

### Can I customize the WebUI?

The WebUI is embedded in the binary. To customize:
1. Clone repository
2. Edit files in `web/` directory
3. Rebuild: `go build -o niac ./cmd/niac`

## Advanced Usage

### Can I script NIAC-Go?

Yes! Use the API:
```python
import requests

token = "your-api-token"
headers = {"Authorization": f"Bearer {token}"}

# Get stats
stats = requests.get("http://localhost:8080/api/v1/stats", headers=headers).json()
print(f"Packets sent: {stats['stack']['packets_sent']}")

# Update config
new_config = {"content": open("new-config.yaml").read()}
requests.put("http://localhost:8080/api/v1/config", json=new_config, headers=headers)
```

### How do I run in Docker?

See `docs/DEPLOYMENT.md` for Docker and Kubernetes deployment guides.

### Can I use NIAC-Go in CI/CD?

Yes! See `docs/CI-CD.md` for GitHub Actions, GitLab CI, and Jenkins examples.

### How do I contribute?

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Submit pull request

See `CONTRIBUTING.md` for detailed guidelines.

## Support

### Where can I get help?

- **Documentation**: https://github.com/krisarmstrong/niac-go/tree/main/docs
- **Issues**: https://github.com/krisarmstrong/niac-go/issues
- **Discussions**: https://github.com/krisarmstrong/niac-go/discussions

### Is there commercial support?

NIAC-Go is open source (MIT license). For commercial support or custom development, contact the maintainers through GitHub.

### How often is NIAC-Go updated?

- **Security patches**: As needed (CRITICAL/HIGH issues)
- **Feature releases**: Monthly minor versions
- **Major releases**: 1-2 per year

See `CHANGELOG.md` for release history.
