# How Error Injection Works

Quick explanation of how to inject network problems into NIAC simulations.

## Three Methods - Choose Based on Your Need

### Method 1: Static Configuration (Structural Problems)

**How it works:** Configure problems directly in .cfg files. Good for permanent structural issues.

**Pros:**
- No special flags needed
- Problems persist across runs
- Perfect for duplicate IP, bad subnet masks
- Simple to configure

**Example - Duplicate IP:**
```
Device(
    IpAddr(192.168.1.10)
    MacAddr(00AA11BB22C1)
    SnmpAgent(
        Include("device_walks/cisco/Switch1.walk")
    )
)

Device(
    IpAddr(192.168.1.10)  // Same IP!
    MacAddr(00AA11BB22C5)  // Different MAC
    SnmpAgent(
        Include("device_walks/cisco/Switch2.walk")
    )
)
```

**Use this method when:**
- You want duplicate IP addresses
- You need bad subnet masks (modify walk file)
- You want half-duplex interfaces (modify walk file)
- The problem is configuration-based, not counter-based

---

### Method 2: Dynamic Errors (Pre-Configured Timeline)

**How it works:** Use `AddMib()` in .cfg files. Errors change over time during simulation.

**Pros:**
- No special flags needed
- Errors are temporary (only exist during simulation)
- Can simulate increasing/decreasing errors
- Original walks never touched
- Perfect for automated testing

**Example:**
```
Device(
    IpAddr(192.168.1.10)
    MacAddr(00AA11BB22C1)
    SnmpAgent(
        // Errors start at 50, increase to 450 over 2 minutes
        AddMib("1.3.6.1.2.1.2.2.1.14.1", "varimib", "(0 50)(6000 150)(12000 450)")

        Include("device_walks/cisco/Switch.walk")  // Original walk untouched!
    )
)
```

**Use this method when:**
- You want errors that change over time automatically
- You're doing automated testing
- You want pre-programmed error scenarios
- You don't need to clear errors manually

---

### Method 3: Interactive Mode (Live Demo Control) ⭐ NEW

**How it works:** Start NIAC with `--interactive` flag. Inject/clear errors in real-time.

**Pros:**
- Real-time control during demos
- Inject errors on-demand
- Clear errors to show recovery
- Per-interface granularity
- Change interface speed/duplex
- Works with any .cfg file

**Cons:**
- Requires `--interactive` flag
- Need to use interactive menu

**Example:**
```bash
sudo ./niac run basic-network --interactive

# NIAC starts with clean network
# Press 'i' to enter interactive mode
# Select device → interface → error type → value
# Monitoring tool shows the error
# Press 'i' again to clear the error
# Monitoring tool shows recovery
```

**Interactive Menu:**
```
╔════════════════════════════════════════════════════════════════╗
║           NIAC Interactive Control Mode v6.0                   ║
╠════════════════════════════════════════════════════════════════╣
║  Running Configuration: scenario_configs/basic-network.cfg    ║
║  Active Devices: 5                                            ║
║  Active Errors: 0                                             ║
╠════════════════════════════════════════════════════════════════╣
║  [1] Inject Error                                             ║
║  [2] Clear Error                                              ║
║  [3] Modify Interface Configuration                           ║
║  [4] Show Active Errors                                       ║
║  [5] Show Devices                                             ║
║  [Q] Quit Interactive Mode (NIAC keeps running)               ║
╚════════════════════════════════════════════════════════════════╝
```

**Use this method when:**
- You're doing live demonstrations
- You want to show problem → alert → fix → recovery
- You need to inject/clear errors on demand
- You want per-interface control
- You need to change interface speed or duplex mode

---

## Are Errors Permanent?

**Method 1 (Static Config):** YES
- Configured in .cfg file
- Exists every time you run that .cfg
- Delete from .cfg to remove

**Method 2 (AddMib):** NO
- Errors only exist during simulation
- Stop simulation = errors gone
- Remove AddMib() from .cfg to disable

**Method 3 (Interactive):** NO
- Errors only exist while you have them active
- Clear error = immediately gone
- Restart NIAC = back to clean state

## Quick Commands

**Run with static/dynamic errors (Methods 1 & 2):**
```bash
sudo ./niac run network-with-errors
```

**Run with interactive mode (Method 3):**
```bash
sudo ./niac run basic-network --interactive
```

**Run pre-made error scenario:**
```bash
sudo ./niac run network-with-errors
```

## Summary

| Question | Static Config | AddMib | Interactive |
|----------|---------------|--------|-------------|
| Modifies walk files? | Sometimes | ❌ Never | ❌ Never |
| Requires special flag? | ❌ No | ❌ No | ✅ --interactive |
| Errors change over time? | ❌ No | ✅ Yes (pre-programmed) | ✅ Yes (on demand) |
| Errors permanent? | ✅ Yes (in .cfg) | ❌ No (temporary) | ❌ No (on demand) |
| Real-time control? | ❌ No | ❌ No | ✅ Yes |
| Per-interface control? | ✅ Yes | ✅ Yes | ✅ Yes |
| Best for? | Structural issues | Automated tests | Live demos |

**Choose Static Config** for duplicate IP, bad subnet masks, permanent configuration problems.

**Choose AddMib** for automated testing with time-based error progression.

**Choose Interactive** for live demonstrations where you control exactly when errors appear and disappear.

## Full Documentation

See `docs/ERROR_INJECTION.md` for complete guide with all OID mappings.

See `docs/INTERACTIVE_MODE_DESIGN.md` for interactive mode architecture.

## Demo Workflow Example (Interactive Mode)

```
Terminal 1: Start NIAC
$ cd demo_configs
$ sudo ./niac run basic-network --interactive

[NIAC] Network running...
[NIAC] Press 'i' for interactive menu

Terminal 2: Open monitoring tool
- Shows all devices healthy

Back to Terminal 1:
Press 'i'
→ [1] Inject Error
→ Select device: 192.168.1.10
→ Select interface: 1 (Fa0/1)
→ Select error type: FCS Errors
→ Enter value: 500
→ ✓ Error Injected

Back to Terminal 2 (monitoring tool):
- Alert: "High FCS errors on 192.168.1.10 interface Fa0/1"
- Demonstrate the problem to your audience

Back to Terminal 1:
Press 'i'
→ [2] Clear Error
→ Select device: 192.168.1.10
→ Select interface: 1
→ Select error type: FCS Errors
→ ✓ Error Cleared

Back to Terminal 2 (monitoring tool):
- Shows problem resolved
- Demo complete!
```

## Supported Error Types

**Interface-Level (Per-Interface):**
- FCS Errors
- Input Errors
- Output Errors
- Input Discards
- Output Discards
- High Utilization

**Device-Level (Whole Device):**
- High CPU
- High Memory
- High Disk

**Interface Configuration (Interactive Mode):**
- Change Speed (10M/100M/1G/2.5G/5G/10G/25G/40G/100G or custom)
- Change Duplex Mode (Half/Full)

**Static Configuration (Any Method):**
- Duplicate IP Address
- Bad Subnet Mask
- Half-Duplex (via walk file)
- Spanning Tree Changes

## Author

Kris Armstrong <kris.armstrong@me.com>
