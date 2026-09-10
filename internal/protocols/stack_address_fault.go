package protocols

import (
	"errors"
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrFaultConflictAbsent means the requested address has no active peer owner on the server's segment.
var ErrFaultConflictAbsent = errors.New("fault address must belong to an active peer on the DHCP server's segment")

// SetDeviceAddressFault arms an explicitly addressed service outcome.
func (s *Stack) SetDeviceAddressFault(deviceTarget string, kind devicestate.DeviceFaultType, address netip.Addr) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(deviceTarget)
	if err != nil {
		return err
	}
	if kind != devicestate.FaultDuplicateDHCPOffer {
		return devicestate.ErrDeviceFaultTypeInvalid
	}
	if s.dhcpHandlers[device] == nil {
		return ErrFaultServiceAbsent
	}
	if !s.duplicateOfferPeer(device, address) {
		return ErrFaultConflictAbsent
	}
	return store.SetDeviceAddressFault(kind, address)
}

// ClearDeviceFault clears one numeric or addressed service outcome.
func (s *Stack) ClearDeviceFault(deviceTarget string, kind devicestate.DeviceFaultType) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	_, store, err := s.interfaceFaultTarget(deviceTarget)
	if err != nil {
		return err
	}
	return store.ClearDeviceFault(kind)
}

func (s *Stack) duplicateOfferPeer(server *config.Device, address netip.Addr) bool {
	if !address.Is4() {
		return false
	}
	if s.fabric != nil {
		endpoint, found := s.fabric.endpointForAddress(address)
		if !found || endpoint.device == server || endpoint.network != s.fabric.attachmentNetwork ||
			!s.fabric.deviceOnAttachment(
				server,
			) || !s.fabric.interfaceAvailable(endpoint.device, endpoint.interfaceName) {
			return false
		}
		for _, network := range s.fabric.topology.Networks {
			if network.Name == endpoint.network {
				return config.ValidIPv4Host(network.Prefix, address)
			}
		}
		return false
	}
	for _, peer := range s.devicesForStateIPv4(s.discoveryVLAN(server), net.IP(address.AsSlice())) {
		if peer == server {
			continue
		}
		for _, iface := range s.deviceStates[peer].Snapshot().Network.Interfaces {
			if iface.AdminUp && iface.OperUp && iface.Address.Addr() == address &&
				config.ValidIPv4Host(iface.Address, address) {
				return true
			}
		}
	}
	return false
}

func (h *DHCPHandler) offerAddress(mac net.HardwareAddr, hostname string) (net.IP, error) {
	if store := h.stack.deviceStates[h.serverDevice]; store != nil {
		address := store.DeviceFaultAddress(devicestate.FaultDuplicateDHCPOffer)
		if address.IsValid() {
			if !h.stack.duplicateOfferPeer(h.serverDevice, address) {
				return nil, ErrFaultConflictAbsent
			}
			return net.IP(address.AsSlice()), nil
		}
	}
	lease, err := h.allocateLease(mac, nil, hostname)
	if err != nil {
		return nil, err
	}
	return lease.IP, nil
}

// Read state before the lease lock: the state observer acquires that lock.
func (h *DHCPHandler) conflictsWithOffer(ip net.IP) bool {
	store := h.stack.deviceStates[h.serverDevice]
	if store == nil || len(ip) == 0 {
		return false
	}
	address := store.DeviceFaultAddress(devicestate.FaultDuplicateDHCPOffer)
	return address.IsValid() && ip.Equal(net.IP(address.AsSlice()))
}
