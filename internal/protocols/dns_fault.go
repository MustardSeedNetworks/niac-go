package protocols

import (
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// dnsQuerySilenced reports whether an armed timeout fault swallows every query
// this server would answer. The client sees no packet at all, which is what a
// resolver timeout is on the wire; answering with SERVFAIL instead would be a
// different, immediately-diagnosable failure.
func (h *DNSHandler) dnsQuerySilenced(serverDevice *config.Device) bool {
	return h.stack.deviceFaultActive(serverDevice, devicestate.FaultDNSTimeout)
}

// applyNXDomainFault rewrites an answered response into an authoritative
// NXDOMAIN. It runs after resolution rather than instead of it so the server
// still behaves like itself: same authority bit, same question section, same
// server address, only the name is now reported as non-existent.
func (h *DNSHandler) applyNXDomainFault(
	response *layers.DNS, serverDevice *config.Device,
) *layers.DNS {
	if !h.stack.deviceFaultActive(serverDevice, devicestate.FaultDNSNXDomain) {
		return response
	}
	response.Answers = []layers.DNSResourceRecord{}
	response.ResponseCode = layers.DNSResponseCodeNXDomain
	return response
}
