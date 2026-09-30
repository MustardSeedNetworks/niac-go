package protocols

import (
	"net"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	// raOriginationTick is how often origination checks for due router
	// advertisements; authored periods are whole seconds.
	raOriginationTick = time.Second
	// defaultRAPeriod is RFC 4861's default MaxRtrAdvInterval, used when a
	// device authors an advertisement without a period.
	defaultRAPeriod = 600 * time.Second

	eui64Marker0 = 0xff
	eui64Marker1 = 0xfe
	// eui64UniversalLocalBit is flipped between a MAC's first octet and a
	// modified EUI-64 interface identifier (RFC 4291 appendix A).
	eui64UniversalLocalBit = 0x02

	allNodesIPv6 = "ff02::1"
	allNodesMAC  = "33:33:00:00:00:01"
)

// Start begins sending unsolicited router advertisements from every device
// that authors one, each at its own period. Safe to call again after Stop.
func (h *ICMPv6Handler) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopChan != nil {
		return
	}
	stop := make(chan struct{})
	h.stopChan = stop

	go func() {
		due := h.sendDueRouterAdvertisements(time.Now(), nil)
		ticker := time.NewTicker(raOriginationTick)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				due = h.sendDueRouterAdvertisements(now, due)
			case <-stop:
				return
			}
		}
	}()
}

// Stop halts router advertisement origination. Safe to call multiple times.
func (h *ICMPv6Handler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopChan != nil {
		close(h.stopChan)
		h.stopChan = nil
	}
}

// sendDueRouterAdvertisements sends an advertisement to all-nodes from each
// device that is due by now and returns when each is next due, scheduling from
// the last due time as STP origination does so tick jitter cannot stretch the
// period.
func (h *ICMPv6Handler) sendDueRouterAdvertisements(
	now time.Time,
	due map[*config.Device]time.Time,
) map[*config.Device]time.Time {
	h.stack.reloadMu.RLock()
	defer h.stack.reloadMu.RUnlock()

	dstIP := net.ParseIP(allNodesIPv6)
	dstMAC, _ := net.ParseMAC(allNodesMAC)
	next := make(map[*config.Device]time.Time, len(due))
	for _, device := range h.stack.AllDevices() {
		if !deviceCanAdvertiseIPv6(device) ||
			(h.stack.fabric != nil && !h.stack.fabric.advertisesAtClient(device)) {
			continue
		}
		at, scheduled := due[device]
		if scheduled && now.Before(at) {
			next[device] = at
			continue
		}
		err := h.sendRouterAdvertisement(h.stack.discoveryVLAN(device), device, dstIP, dstMAC)
		if err != nil && h.debugLevel.Load() >= int32(DebugLevelInfo) {
			logging.Debugf("ICMPv6: unsolicited RA from %s not sent: %v", device.Name, err)
		}
		period := routerAdvertisementPeriod(device)
		if !scheduled || now.Sub(at) >= period {
			at = now
		}
		next[device] = at.Add(period)
	}
	return next
}

func routerAdvertisementPeriod(device *config.Device) time.Duration {
	if seconds := getRAConfig(device).Period; seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultRAPeriod
}

// linkLocalAddress is the address a device sends Neighbor Discovery from: an
// authored fe80::/10 address, else the modified EUI-64 address of its MAC. A
// host discards a router advertisement from any other source (RFC 4861
// section 6.1.2).
func linkLocalAddress(device *config.Device) net.IP {
	for _, ip := range device.IPAddresses {
		if ip.To4() == nil && ip.IsLinkLocalUnicast() {
			return ip
		}
	}
	mac := device.MACAddress
	if len(mac) != SizeOfMac {
		return nil
	}
	return net.IP{
		0xfe, 0x80, 0, 0, 0, 0, 0, 0,
		mac[0] ^ eui64UniversalLocalBit, mac[1], mac[2], eui64Marker0,
		eui64Marker1, mac[3], mac[4], mac[5],
	}
}

// devicesAtLinkLocal resolves a link-local Neighbor Solicitation target that no
// device authored: the derived address encodes the MAC, so the device that owns
// it is found by MAC and answers only if it speaks IPv6 at all.
func devicesAtLinkLocal(table *DeviceTable, target net.IP) []*config.Device {
	ip := target.To16()
	if ip == nil || !ip.IsLinkLocalUnicast() || ip[11] != eui64Marker0 || ip[12] != eui64Marker1 {
		return nil
	}
	mac := net.HardwareAddr{ip[8] ^ eui64UniversalLocalBit, ip[9], ip[10], ip[13], ip[14], ip[15]}
	device := table.GetByMAC(mac)
	if device == nil || firstIPv6Address(device) == nil || !linkLocalAddress(device).Equal(ip) {
		return nil
	}
	return []*config.Device{device}
}
