# Platform Support Guide

This document describes platform-specific requirements, capabilities, and limitations for NIAC-Go.

## Platform Support Matrix

| Feature | Linux | macOS | Windows |
| --------- | ------- | ------- | --------- |
| LLDP Discovery | Full | Full | Full |
| CDP Discovery | Full | Full | Full |
| ARP Scanning | Full | Full | Full |
| NDP (IPv6) | Full | Full | Full |
| Raw Packet Capture | Full | Full | Full* |
| Performance | Best | Good | Good |
| Privilege Required | root/CAP_NET_RAW | root | Administrator |

*Windows requires Npcap installation

---

## Linux

Linux provides the best performance and most complete feature support.

### Requirements

- **Minimum OS**: Ubuntu 20.04, RHEL 8, Debian 11, or equivalent
- **Architecture**: x86_64, ARM64
- **Go Version**: 1.27.0+
- **Dependencies**: libpcap-dev

### Installation

```bash
# Ubuntu/Debian
sudo apt-get install libpcap-dev

# RHEL/CentOS/Fedora
sudo dnf install libpcap-devel

# Build
go build -o niac ./cmd/niac
```

### Privilege Requirements

NIAC requires raw socket access for packet capture. Options:

1. **Run as root** (simplest):

   ```bash
   sudo ./niac
   ```

2. **Use capabilities** (recommended for production):

   ```bash
   sudo setcap cap_net_raw,cap_net_admin=eip ./niac
   ./niac  # No sudo needed
   ```

### Performance Notes

- Linux uses AF_PACKET sockets for optimal performance
- Best throughput on high-speed networks (10GbE+)
- Lower latency than other platforms for protocol discovery

---

## macOS

macOS provides full functionality with some performance considerations.

### Requirements

- **Minimum OS**: macOS 10.15 (Catalina) or later
- **Architecture**: x86_64 (Intel), ARM64 (Apple Silicon)
- **Go Version**: 1.27.0+
- **Dependencies**: libpcap (included with Xcode Command Line Tools)

### Installation

```bash
# Install Xcode Command Line Tools (includes libpcap)
xcode-select --install

# Build
go build -o niac ./cmd/niac
```

### Privilege Requirements

macOS uses BPF (Berkeley Packet Filter) for packet capture:

```bash
# Run with sudo
sudo ./niac
```

### Known Limitations

1. **BPF Device Access**: Requires root privileges; no capability-based alternative
2. **Performance**: Slightly higher latency than Linux due to BPF overhead
3. **Interface Names**: Use `en0`, `en1` format (not `eth0`)

### Apple Silicon Notes

- Native ARM64 builds provide best performance on M1/M2/M3 Macs
- Rosetta 2 (x86_64 emulation) works but with performance penalty
- Build natively: `GOARCH=arm64 go build -o niac ./cmd/niac`

---

## Windows

Windows provides full functionality with Npcap dependency.

### Requirements

- **Minimum OS**: Windows 10 version 1903 or later, Windows 11
- **Architecture**: x86_64 (AMD64)
- **Go Version**: 1.27.0+
- **Dependencies**: Npcap (https://npcap.com)

### Installation

1. **Install Npcap**:
   - Download from https://npcap.com/#download
   - Run installer with **"Install Npcap in WinPcap API-compatible Mode"** checked
   - Reboot if prompted

2. **Install a release archive** from an elevated PowerShell session:

   ```powershell
   Expand-Archive niac-<version>-windows-<arch>.zip -DestinationPath .
   cd niac-<version>-windows-<arch>
   .\install.ps1
   ```

3. **Or build from source**:

   ```powershell
   go build -o niac.exe ./cmd/niac
   ```

### Privilege Requirements

Run as Administrator:

- Right-click Command Prompt/PowerShell → "Run as administrator"
- Or use elevated terminal in Windows Terminal

```powershell
# Run NiAC
.\niac.exe
```

### Known Limitations

1. **Npcap Required**: WinPcap is deprecated; use Npcap
2. **Interface Names**: Use adapter names from `ipconfig /all` or NIAC's interface list
3. **Firewall**: Windows Defender Firewall may block some traffic; add exceptions if needed
4. **Performance**: Good but slightly behind Linux on high-speed networks

### Windows 11 Specific Notes

- Npcap 1.60+ recommended for Windows 11 compatibility
- Some older NICs may have driver compatibility issues
- Virtual adapters (Hyper-V, WSL) may not support all capture features

### Troubleshooting

**"No interfaces found" error:**

1. Verify Npcap is installed: Check Programs and Features
2. Reinstall Npcap with WinPcap compatibility mode
3. Run as Administrator

**"Access denied" error:**

1. Ensure running as Administrator
2. Check Windows Defender Firewall settings
3. Temporarily disable antivirus to test

---

## Release Architecture Support

| Architecture | Linux | macOS | Windows |
| -------------- | ------- | ------- | --------- |
| x86_64 (AMD64) | ✓ | - | ✓ |
| ARM64 | ✓ | ✓ (Apple Silicon) | ✓ |

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o niac-linux-amd64 ./cmd/niac

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o niac-linux-arm64 ./cmd/niac

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o niac-darwin-arm64 ./cmd/niac

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o niac-windows-amd64.exe ./cmd/niac
```

---

## Feature Comparison by Platform

### Protocol Support

All protocols are supported on all platforms:

- LLDP (Link Layer Discovery Protocol)
- CDP (Cisco Discovery Protocol)
- ARP (Address Resolution Protocol)
- NDP (Neighbor Discovery Protocol for IPv6)

### Performance Characteristics

| Metric | Linux | macOS | Windows |
| -------- | ------- | ------- | --------- |
| Packet capture latency | Lowest | Low | Medium |
| High-speed capture (10GbE) | Excellent | Good | Good |
| CPU efficiency | Best | Good | Good |
| Memory usage | Lowest | Low | Medium |

### Debug Features

Debug verbosity is one global level and behaves the same on every platform:

```bash
sudo niac daemon --once -d 3 en0 config.yaml   # 0 quiet .. 3 trace
```

---

## Release Artifacts

GitHub Actions owns every release build. GoReleaser Cross produces Linux and
Apple Silicon macOS artifacts on Linux; native GitHub Windows runners produce
the CGO-enabled Windows artifacts. Local builds are development-only, and Intel
macOS is not a release target. GitHub Releases is the canonical distribution
channel.

---

## Getting Help

For platform-specific issues:

1. Check this document first
2. Review `docs/TROUBLESHOOTING.md`
3. Run with `--debug` flags to gather diagnostic info
4. Report issues at: https://github.com/MustardSeedNetworks/niac-go/issues

Include in bug reports:

- Platform and OS version
- NIAC version (`niac --version`)
- Go version (`go version`)
- Npcap version (Windows only)
- Full error message and debug output
