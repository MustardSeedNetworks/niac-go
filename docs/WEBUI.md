# WebUI Documentation

## Overview

The NIAC-Go WebUI provides a web-based interface for monitoring and controlling network simulations. It's a React-based single-page application embedded in the binary.

## Accessing the WebUI

### Starting the API Server

```bash
# Generate API token
export NIAC_API_TOKEN=$(openssl rand -base64 32)

# Start with API server
sudo niac run config.yaml --api :8080

# Open browser to http://localhost:8080
```

### Authentication

1. On first visit, you'll be prompted for the API token
2. Enter the `NIAC_API_TOKEN` value
3. Token is stored in browser localStorage
4. Click "Connect" to authenticate

## Main Dashboard

### Statistics Overview

The main dashboard shows real-time statistics:

**Packet Counters:**
- **Packets Sent**: Total packets transmitted
- **Packets Received**: Total packets received
- **Errors**: Protocol processing errors

**Protocol Statistics:**
- **ARP**: Requests and replies
- **ICMP**: Ping requests and replies
- **DNS**: Query count
- **DHCP**: DHCP requests processed
- **SNMP**: SNMP queries handled

**System Information:**
- **Version**: NIAC-Go version
- **Interface**: Network interface in use
- **Device Count**: Number of simulated devices
- **Goroutines**: Current goroutine count (for leak detection)
- **Uptime**: Time since simulation started

### Auto-Refresh

- Statistics refresh every 5 seconds automatically
- Pause/resume auto-refresh with the toggle button
- Manual refresh available

## Device List

View all configured devices with:

**Device Information:**
- Device name
- Device type (router, switch, server, etc.)
- IP addresses
- Supported protocols

**Actions:**
- Click device name to view details
- Filter by device type
- Search by name or IP

## Configuration Management

### View Configuration

- **GET /api/v1/config** - View current YAML configuration
- Syntax highlighting for readability
- Download configuration file

### Update Configuration

1. Click "Edit Configuration"
2. Modify YAML in editor
3. Validation happens on submit
4. Click "Save" to apply changes
5. Configuration updates require CSRF token (automatically handled)

**Note**: Configuration changes may require simulation restart for some settings.

## PCAP Replay

### Upload and Replay PCAP Files

1. Navigate to "Replay" tab
2. Click "Upload PCAP"
3. Select file (max 100MB)
4. Set replay options:
   - **Speed**: Playback speed multiplier (0.1x to 10x)
   - **Loop**: Repeat playback continuously

5. Click "Start Replay"

### Replay Controls

- **Play/Pause**: Control playback
- **Stop**: End replay session
- **Progress**: View packets sent vs total
- **Status**: Current replay state

### Replay from File Path

If PCAP is on server:
1. Enter file path
2. Set speed/loop options
3. Click "Start Replay"

## Alerts Configuration

Configure threshold-based alerting:

**Settings:**
- **Packets Threshold**: Alert when packet count exceeds value
- **Webhook URL**: HTTP endpoint for alerts (optional)

**Webhook Payload:**
```json
{
  "alert": "Packet threshold exceeded",
  "packets": 100000,
  "threshold": 50000,
  "timestamp": "2025-11-14T12:34:56Z"
}
```

## Network Topology

### Topology Visualization

- View network topology graph
- Nodes represent devices
- Edges show connections (based on ARP/LLDP/CDP)

### Export Options

**Formats:**
- **JSON**: Machine-readable topology data
- **GraphML**: For import into yEd, Gephi
- **DOT**: For Graphviz rendering

**Example:**
```bash
# Export to GraphML
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/topology/export?format=graphml" \
  -o topology.graphml
```

## Error Logs

View recent protocol errors:

**Error Information:**
- Timestamp
- Protocol (DNS, DHCP, SNMP, etc.)
- Error message
- Count (if repeated)

**Filters:**
- By protocol
- By time range
- Search error messages

## Runtime Information

**System Stats:**
- Go version
- OS/Architecture
- Memory usage (if available)
- Goroutine count
- Start time / uptime

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+K` | Focus search |
| `r` | Manual refresh |
| `p` | Pause/resume auto-refresh |
| `/` | Focus device filter |
| `Esc` | Close modals |

## Browser Support

**Fully Supported:**
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

**Partially Supported:**
- IE 11 (degraded experience)

## Performance

### Recommendations for Large Deployments

**100+ Devices:**
- Auto-refresh may slow down
- Consider increasing refresh interval
- Use device filtering

**1000+ Devices:**
- Virtual scrolling being developed (#126)
- WebUI performance optimization planned (#125)
- Currently may experience lag

### Optimization Tips

1. **Disable auto-refresh** when not actively monitoring
2. **Use filters** to reduce displayed devices
3. **Close unused tabs** in browser
4. **Use API directly** for automated monitoring

## Troubleshooting

### Cannot Connect to API

**Symptoms:** "Failed to fetch" or connection errors

**Solutions:**
1. Verify API server is running: `curl http://localhost:8080/api/v1/version`
2. Check firewall rules
3. Ensure correct port (default :8080)
4. Try localhost vs 127.0.0.1 vs machine IP

### Authentication Fails

**Symptoms:** 401 Unauthorized errors

**Solutions:**
1. Verify `NIAC_API_TOKEN` is set correctly
2. Check token was copied completely (no whitespace)
3. Clear browser localStorage and re-enter token
4. Check server logs for token mismatch

### CSRF Token Errors

**Symptoms:** 403 Forbidden with "csrf_token_missing"

**Solutions:**
1. Refresh page to get new CSRF token
2. Check browser console for errors
3. Clear cache and reload
4. Ensure cookies/localStorage enabled

### Slow Performance

**Symptoms:** Laggy UI, slow updates

**Solutions:**
1. Reduce number of displayed devices
2. Increase auto-refresh interval
3. Disable auto-refresh when not needed
4. Check goroutine count for leaks
5. Monitor server CPU/memory

### WebUI Not Loading

**Symptoms:** Blank page or 404 errors

**Solutions:**
1. Verify you're accessing correct URL
2. Check SPA serving is enabled (default)
3. Clear browser cache
4. Try incognito/private mode
5. Check browser console for errors

## Development

### Building WebUI from Source

The WebUI is embedded in the Go binary. To customize:

```bash
# Clone repository
git clone https://github.com/krisarmstrong/niac-go.git
cd niac-go

# Install Node.js dependencies (if modifying WebUI)
cd web && npm install

# Build WebUI
npm run build

# Build Go binary (embeds WebUI)
cd .. && go build -o niac ./cmd/niac
```

### WebUI Stack

- **Framework**: React 18
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **State**: React Hooks (useState, useEffect)
- **HTTP Client**: Fetch API
- **Charts**: Chart.js (for statistics)

### File Structure

```
web/
├── src/
│   ├── App.jsx          # Main application component
│   ├── components/      # React components
│   ├── api/             # API client
│   └── utils/           # Utility functions
├── public/              # Static assets
├── index.html           # Entry point
└── vite.config.js       # Build configuration
```

## Future Enhancements

Planned improvements (see GitHub issues):

- **#125**: React re-render optimization for better performance
- **#126**: Virtual scrolling for device lists (1000+ devices)
- **#133**: State management library (Redux/Zustand)
- Enhanced topology visualization
- Real-time log streaming
- Packet hex dump viewer
- Configuration wizard

## API Integration

The WebUI uses the REST API documented in [API-EXAMPLES.md](API-EXAMPLES.md).

**Key Endpoints:**
- `GET /api/v1/stats` - Statistics (polled every 5s)
- `GET /api/v1/devices` - Device list
- `GET /api/v1/config` - Configuration
- `PUT /api/v1/config` - Update configuration (requires CSRF)
- `POST /api/v1/replay` - Start replay (requires CSRF)

**CSRF Protection:**
1. WebUI fetches CSRF token on load: `GET /api/v1/csrf-token`
2. Token included in all POST/PUT/PATCH/DELETE requests
3. Token refreshed automatically on 403 errors

## Customization

### Changing Refresh Intervals

The WebUI uses tiered polling intervals optimized for different data types:

```typescript
const POLL_INTERVALS = {
  FAST: 2000,      // 2s - Real-time simulation status
  MEDIUM: 5000,    // 5s - Live stats
  SLOW: 15000,     // 15s - Historical data
  VERY_SLOW: 60000, // 1m - Static data like version
}
```

**To customize:**

1. Edit `webui/src/App.tsx`
2. Modify the `POLL_INTERVALS` constant (lines 136-141)
3. Rebuild WebUI: `cd webui && npm run build`
4. Rebuild Go binary: `go build -o niac ./cmd/niac`

**Examples:**
- Reduce network traffic: Increase all intervals (e.g., MEDIUM: 10000)
- Real-time monitoring: Decrease MEDIUM to 2000 or 3000
- Battery/bandwidth saving: Set all to VERY_SLOW intervals

**Note**: UI-based polling configuration planned for future release

### Custom Branding

To add custom branding:

1. Replace `web/public/logo.png`
2. Edit `web/index.html` title
3. Modify `web/src/App.jsx` header
4. Rebuild

### Dark Mode

Dark mode support is planned but not yet implemented. Current theme is light only.

## Security Considerations

### Authentication

- API token required for all operations
- Token stored in browser localStorage
- No session management (stateless)
- Token transmitted in Authorization header

### CSRF Protection

- All state-changing operations require CSRF token
- Token unique per server instance
- Token regenerated on server restart
- Prevents cross-site request forgery

### HTTPS

WebUI does not handle TLS directly. For production:

1. Use reverse proxy (nginx, Apache, Caddy)
2. Configure TLS termination at proxy
3. Proxy to NIAC-Go API server

**Example nginx config:**
```nginx
server {
    listen 443 ssl http2;
    server_name niac.example.com;

    ssl_certificate /etc/ssl/certs/niac.crt;
    ssl_certificate_key /etc/ssl/private/niac.key;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## Support

For WebUI issues:
- Check browser console for errors
- Review server logs
- Create GitHub issue with screenshots
- Include browser version and OS

## Screenshots

*Note: Screenshots will be added in future update. For now, start the WebUI to explore features.*

Key screens to explore:
1. Dashboard - Statistics overview
2. Devices - Device list and details
3. Configuration - YAML editor
4. Replay - PCAP upload and playback
5. Topology - Network visualization
6. Alerts - Threshold configuration
