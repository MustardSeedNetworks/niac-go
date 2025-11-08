# Code Review Findings

## 1. DHCP broadcast traffic never reaches the server (High)
- **Location**: `pkg/protocols/ip.go:30-74`, `pkg/protocols/device_table.go:32-54`
- **Issue**: `IPHandler` unconditionally drops any packet whose destination IP is not already registered in the device table. Broadcast DHCP DISCOVER/REQUEST frames are sent to `255.255.255.255`, which is never added to the table, so `len(devices)==0` and the function returns before the UDP handler sees the payload. As a result the DHCP handler is never invoked and clients can never obtain leases.
- **Impact**: All DHCP flows silently fail on the very first packet, even though the CLI advertises DHCP support. This is an externally visible regression introduced by the Go rewrite and makes the DHCP feature unusable.
- **Recommendation**: Treat broadcast and multicast destinations as eligible traffic (e.g., explicitly allow IPv4 broadcast, subnet-directed broadcast, and ff02:: multicast) and route them to the relevant handlers. Alternatively, register synthetic broadcast addresses in `DeviceTable` during initialization, but explicit checks in `IPHandler` make the intent clearer.

## 2. DHCP handler cannot lease addresses even if invoked (High)
- **Location**: `cmd/niac/main.go:474-514`, `pkg/protocols/dhcp.go:78-188`
- **Issue**: `configureServiceHandlers` never configures an IP pool on the DHCP handler (`SetPool` is never called) and the parsed static leases are not used anywhere. `allocateLease` therefore always finds `h.ipPool` empty and returns `"no available IP addresses"`, meaning every OFFER/ACK is skipped. Even environments with static lease definitions cannot work because that data is never consulted.
- **Impact**: DHCP support is effectively dead code. Users will see no responses even if issue #1 is fixed, which makes troubleshooting very difficult.
- **Recommendation**: Honor the YAML DHCP pool/lease definitions by seeding `DHCPHandler` with an address pool (e.g., derive it from config or explicit range) and look up static `ClientLeases` before falling back to dynamic allocation.

## 3. Ctrl+C hangs on idle interfaces because capture reads block forever (High)
- **Location**: `pkg/capture/capture.go:21-46`, `pkg/protocols/stack.go:189-241`, `cmd/niac/main.go:409-415`
- **Issue**: The capture engine opens the interface with `pcap.BlockForever` and `receiveThread` immediately calls `ReadPacket` inside the `default` branch of a `select`. When `Stack.Stop` is invoked (e.g., on SIGINT), it closes `stopChan` and waits on `wg`, but the receive goroutine is still blocked inside libpcap until another frame arrives. Because the capture handle is only closed by the deferred `engine.Close()` *after* `stack.Stop()` returns, shutdown can hang indefinitely on a quiet network.
- **Impact**: Operators cannot stop the simulator cleanly unless traffic happens to arrive. This is a serious operability and UX issue for production use.
- **Recommendation**: Either open the handle with a finite read timeout, or call `s.capture.Close()` before waiting on `wg` so that `ReadPacket` unblocks immediately. Reworking the `select` to check `stopChan` before each read would also help.

## 4. Device simulator cannot be restarted after a Stop (Medium)
- **Location**: `pkg/device/simulator.go:16-83`, `pkg/device/simulator.go:160-183`
- **Issue**: `Simulator` allocates `stopChan` once in the constructor and closes it inside `Stop`. A closed channel cannot be reused, but `Start` does not recreate it, so any subsequent `Start` call spawns goroutines that immediately exit (the select wakes because the channel is closed). This prevents reuse in long-running processes and breaks unit tests that expect start/stop symmetry.
- **Recommendation**: Reinitialize `stopChan` in `Start` (after taking the lock) or use a context that is recreated on each run.

## 5. SNMP traps always leak the "public" community string (Medium)
- **Location**: `pkg/snmp/traps.go:43-82`
- **Issue**: Every trap receiver is instantiated with `Community: "public"` and there is no way to override it through `TrapConfig`. This ignores device-level SNMP configuration and makes it impossible to model production deployments that rely on custom community strings. It also represents a security regression because traps will always be sent with the default community regardless of user intent.
- **Recommendation**: Extend `TrapConfig` to accept a community (and potentially per-receiver credentials) and wire it through `NewTrapSender`. At minimum, pull the value from the parent device's `SNMPConfig` instead of hard-coding `public`.
