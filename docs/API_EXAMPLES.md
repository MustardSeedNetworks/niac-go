# API Usage Examples

This document provides practical examples for interacting with the NIAC-Go API using various tools and programming languages.

## Table of Contents

- [Authentication](#authentication)
- [curl Examples](#curl-examples)
- [Python Examples](#python-examples)
- [JavaScript Examples](#javascript-examples)
- [Go Examples](#go-examples)
- [PowerShell Examples](#powershell-examples)

## Authentication

All API requests (except `/metrics`) require authentication via Bearer token:

```bash
# Generate secure token
export NIAC_API_TOKEN=$(openssl rand -base64 32)

# Start API server
sudo niac run config.yaml --api :8080
```

For state-changing operations (POST/PUT/PATCH/DELETE), you also need a CSRF token:

```bash
# Get CSRF token
CSRF_TOKEN=$(curl -s -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/csrf-token | jq -r '.token')
```

## curl Examples

### Get Statistics

```bash
curl -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/stats | jq .
```

**Response:**
```json
{
  "timestamp": "2025-11-14T12:34:56Z",
  "interface": "eth0",
  "version": "2.6.0",
  "device_count": 5,
  "goroutines": 42,
  "stack": {
    "packets_sent": 12543,
    "packets_received": 10234,
    "arp_requests": 523,
    "arp_replies": 498,
    "icmp_requests": 1024,
    "icmp_replies": 1019,
    "dns_queries": 234,
    "dhcp_requests": 45,
    "snmp_queries": 89,
    "errors": 3
  }
}
```

### List Devices

```bash
curl -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/devices | jq .
```

**Response:**
```json
[
  {
    "name": "router1",
    "type": "router",
    "ips": ["192.168.1.1", "10.0.0.1"],
    "protocols": ["SNMP", "DHCP", "DNS", "HTTP", "LLDP"]
  },
  {
    "name": "switch1",
    "type": "switch",
    "ips": ["192.168.1.2"],
    "protocols": ["SNMP", "LLDP", "CDP"]
  }
]
```

### Get Configuration

```bash
curl -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/config | jq .
```

### Update Configuration

```bash
# Get CSRF token first
CSRF_TOKEN=$(curl -s -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/csrf-token | jq -r '.token')

# Update config
curl -X PUT \
  -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "devices:\n  - name: router1\n    type: router\n    ip_addresses:\n      - 192.168.1.1"
  }' \
  http://localhost:8080/api/v1/config
```

### Start PCAP Replay

```bash
# Get CSRF token
CSRF_TOKEN=$(curl -s -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/csrf-token | jq -r '.token')

# Start replay from file
curl -X POST \
  -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "file": "/path/to/capture.pcap",
    "speed": 1.0,
    "loop": false
  }' \
  http://localhost:8080/api/v1/replay
```

### Upload and Replay PCAP

```bash
# Base64 encode PCAP file
PCAP_DATA=$(base64 -w 0 capture.pcap)

# Get CSRF token
CSRF_TOKEN=$(curl -s -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/csrf-token | jq -r '.token')

# Upload and replay
curl -X POST \
  -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"file\": \"uploaded.pcap\",
    \"data\": \"$PCAP_DATA\",
    \"speed\": 1.0
  }" \
  http://localhost:8080/api/v1/replay
```

### Stop PCAP Replay

```bash
CSRF_TOKEN=$(curl -s -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/csrf-token | jq -r '.token')

curl -X DELETE \
  -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  http://localhost:8080/api/v1/replay
```

### Configure Alerts

```bash
CSRF_TOKEN=$(curl -s -H "Authorization: Bearer $NIAC_API_TOKEN" \
  http://localhost:8080/api/v1/csrf-token | jq -r '.token')

curl -X PUT \
  -H "Authorization: Bearer $NIAC_API_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "packets_threshold": 100000,
    "webhook_url": "http://alertmanager:9093/api/v1/alerts"
  }' \
  http://localhost:8080/api/v1/alerts
```

## Python Examples

### Basic Stats Monitoring

```python
import requests
import time

# Configuration
API_URL = "http://localhost:8080"
TOKEN = "your-api-token-here"

headers = {
    "Authorization": f"Bearer {TOKEN}"
}

def get_stats():
    """Get current statistics"""
    response = requests.get(f"{API_URL}/api/v1/stats", headers=headers)
    response.raise_for_status()
    return response.json()

def main():
    """Monitor statistics every 5 seconds"""
    while True:
        stats = get_stats()
        print(f"Packets: sent={stats['stack']['packets_sent']}, "
              f"received={stats['stack']['packets_received']}, "
              f"errors={stats['stack']['errors']}, "
              f"goroutines={stats['goroutines']}")
        time.sleep(5)

if __name__ == "__main__":
    main()
```

### Update Configuration

```python
import requests

API_URL = "http://localhost:8080"
TOKEN = "your-api-token-here"

headers = {
    "Authorization": f"Bearer {TOKEN}"
}

def get_csrf_token():
    """Get CSRF token for state-changing operations"""
    response = requests.get(f"{API_URL}/api/v1/csrf-token", headers=headers)
    response.raise_for_status()
    return response.json()["token"]

def update_config(yaml_content):
    """Update NIAC configuration"""
    csrf_token = get_csrf_token()

    headers_with_csrf = headers.copy()
    headers_with_csrf["X-CSRF-Token"] = csrf_token
    headers_with_csrf["Content-Type"] = "application/json"

    payload = {"content": yaml_content}

    response = requests.put(
        f"{API_URL}/api/v1/config",
        headers=headers_with_csrf,
        json=payload
    )
    response.raise_for_status()
    return response.json()

# Example usage
new_config = """
devices:
  - name: router1
    type: router
    ip_addresses:
      - 192.168.1.1
    dns_config:
      enabled: true
      records:
        - name: example.com
          type: A
          value: 192.168.1.100
"""

result = update_config(new_config)
print("Configuration updated successfully")
```

### PCAP Replay Automation

```python
import requests
import base64
import time

API_URL = "http://localhost:8080"
TOKEN = "your-api-token-here"

headers = {
    "Authorization": f"Bearer {TOKEN}"
}

def get_csrf_token():
    response = requests.get(f"{API_URL}/api/v1/csrf-token", headers=headers)
    return response.json()["token"]

def upload_and_replay_pcap(pcap_file_path, speed=1.0, loop=False):
    """Upload PCAP file and start replay"""
    # Read and encode PCAP file
    with open(pcap_file_path, 'rb') as f:
        pcap_data = base64.b64encode(f.read()).decode('utf-8')

    csrf_token = get_csrf_token()
    headers_with_csrf = headers.copy()
    headers_with_csrf["X-CSRF-Token"] = csrf_token

    payload = {
        "file": "uploaded.pcap",
        "data": pcap_data,
        "speed": speed,
        "loop": loop
    }

    response = requests.post(
        f"{API_URL}/api/v1/replay",
        headers=headers_with_csrf,
        json=payload
    )
    response.raise_for_status()
    return response.json()

def get_replay_status():
    """Get current replay status"""
    response = requests.get(f"{API_URL}/api/v1/replay", headers=headers)
    response.raise_for_status()
    return response.json()

def stop_replay():
    """Stop current replay"""
    csrf_token = get_csrf_token()
    headers_with_csrf = headers.copy()
    headers_with_csrf["X-CSRF-Token"] = csrf_token

    response = requests.delete(
        f"{API_URL}/api/v1/replay",
        headers=headers_with_csrf
    )
    response.raise_for_status()
    return response.json()

# Example: Replay PCAP at 2x speed
print("Starting replay...")
state = upload_and_replay_pcap("capture.pcap", speed=2.0)
print(f"Replay started: {state}")

# Monitor replay progress
while True:
    status = get_replay_status()
    if not status.get("running", False):
        break
    print(f"Progress: {status.get('packets_sent', 0)} packets sent")
    time.sleep(1)

print("Replay completed")
```

### Prometheus Metrics Integration

```python
import requests
from prometheus_client.parser import text_string_to_metric_families

def fetch_metrics():
    """Fetch and parse Prometheus metrics"""
    response = requests.get("http://localhost:9090/metrics")
    response.raise_for_status()

    for family in text_string_to_metric_families(response.text):
        for sample in family.samples:
            print(f"{sample.name}{sample.labels}: {sample.value}")

fetch_metrics()
```

## JavaScript Examples

### Fetch API (Browser/Node.js)

```javascript
const API_URL = 'http://localhost:8080';
const TOKEN = 'your-api-token-here';

const headers = {
  'Authorization': `Bearer ${TOKEN}`,
  'Content-Type': 'application/json'
};

// Get statistics
async function getStats() {
  const response = await fetch(`${API_URL}/api/v1/stats`, { headers });
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return await response.json();
}

// Get CSRF token
async function getCsrfToken() {
  const response = await fetch(`${API_URL}/api/v1/csrf-token`, { headers });
  const data = await response.json();
  return data.token;
}

// Update configuration
async function updateConfig(yamlContent) {
  const csrfToken = await getCsrfToken();

  const response = await fetch(`${API_URL}/api/v1/config`, {
    method: 'PUT',
    headers: {
      ...headers,
      'X-CSRF-Token': csrfToken
    },
    body: JSON.stringify({ content: yamlContent })
  });

  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return await response.json();
}

// Usage
(async () => {
  try {
    const stats = await getStats();
    console.log('Packets sent:', stats.stack.packets_sent);
    console.log('Goroutines:', stats.goroutines);
  } catch (error) {
    console.error('Error:', error);
  }
})();
```

### Axios (Node.js)

```javascript
const axios = require('axios');

const api = axios.create({
  baseURL: 'http://localhost:8080',
  headers: {
    'Authorization': `Bearer ${process.env.NIAC_API_TOKEN}`
  }
});

async function getCsrfToken() {
  const response = await api.get('/api/v1/csrf-token');
  return response.data.token;
}

async function getDevices() {
  const response = await api.get('/api/v1/devices');
  return response.data;
}

async function startReplay(pcapPath, speed = 1.0) {
  const csrfToken = await getCsrfToken();

  const response = await api.post('/api/v1/replay', {
    file: pcapPath,
    speed: speed,
    loop: false
  }, {
    headers: { 'X-CSRF-Token': csrfToken }
  });

  return response.data;
}

// Usage
(async () => {
  const devices = await getDevices();
  console.log(`Found ${devices.length} devices`);
  devices.forEach(d => console.log(`- ${d.name} (${d.type}): ${d.ips.join(', ')}`));
})();
```

## Go Examples

### Simple Stats Client

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Stats struct {
	Timestamp   string `json:"timestamp"`
	Interface   string `json:"interface"`
	Version     string `json:"version"`
	DeviceCount int    `json:"device_count"`
	Goroutines  int    `json:"goroutines"`
	Stack       struct {
		PacketsSent     uint64 `json:"packets_sent"`
		PacketsReceived uint64 `json:"packets_received"`
		Errors          uint64 `json:"errors"`
	} `json:"stack"`
}

func main() {
	token := os.Getenv("NIAC_API_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "NIAC_API_TOKEN not set")
		os.Exit(1)
	}

	req, _ := http.NewRequest("GET", "http://localhost:8080/api/v1/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var stats Stats
	json.NewDecoder(resp.Body).Decode(&stats)

	fmt.Printf("Version: %s\n", stats.Version)
	fmt.Printf("Devices: %d\n", stats.DeviceCount)
	fmt.Printf("Goroutines: %d\n", stats.Goroutines)
	fmt.Printf("Packets: sent=%d, received=%d, errors=%d\n",
		stats.Stack.PacketsSent, stats.Stack.PacketsReceived, stats.Stack.Errors)
}
```

## PowerShell Examples

### Get Statistics

```powershell
$token = $env:NIAC_API_TOKEN
$headers = @{
    "Authorization" = "Bearer $token"
}

$stats = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/stats" -Headers $headers
Write-Host "Packets Sent: $($stats.stack.packets_sent)"
Write-Host "Packets Received: $($stats.stack.packets_received)"
Write-Host "Goroutines: $($stats.goroutines)"
```

### Update Configuration

```powershell
$token = $env:NIAC_API_TOKEN
$headers = @{
    "Authorization" = "Bearer $token"
}

# Get CSRF token
$csrfResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/csrf-token" -Headers $headers
$csrfToken = $csrfResponse.token

# Update config
$headersWithCsrf = $headers + @{
    "X-CSRF-Token" = $csrfToken
    "Content-Type" = "application/json"
}

$configContent = Get-Content -Path "config.yaml" -Raw
$body = @{
    content = $configContent
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/config" `
    -Method Put `
    -Headers $headersWithCsrf `
    -Body $body
```

## Error Handling

All API errors return standardized JSON responses:

```json
{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded. Please try again later.",
  "timestamp": "2025-11-14T12:34:56Z",
  "path": "/api/v1/stats",
  "method": "GET"
}
```

Common error codes:
- `401`: `unauthorized` - Invalid or missing authentication token
- `403`: `csrf_token_missing` or `csrf_token_invalid` - Missing/invalid CSRF token
- `429`: `rate_limit_exceeded` - Too many requests
- `503`: `replay_unavailable` - Replay engine not available

## Rate Limiting

The API enforces rate limiting:
- **100 requests per second per IP**
- **Burst of 200 requests**

When rate limited, you'll receive HTTP 429 with retry information in the `X-Request-ID` header for tracing.

## Request Tracing

All API responses include an `X-Request-ID` header for debugging:

```bash
curl -v -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats 2>&1 | grep X-Request-ID
# X-Request-ID: a1b2c3d4e5f67890a1b2c3d4e5f67890
```

Use this ID when reporting issues or checking logs.
