package protocols

import (
	"errors"
	"net"
	"net/netip"
)

var errHostNeighborQueueFull = errors.New("host neighbor resolution queue is full")

func (s *Stack) prepareHostEgress(packet *Packet) (*Packet, error) {
	if packet == nil || packet.generatedHost == nil {
		return packet, nil
	}
	s.reloadMu.RLock()
	route, handled, err := s.hostPacketRoute(packet)
	sender, ok := s.notifications.sender.(*stackDatagramSender)
	var mac net.HardwareAddr
	vlan := packet.VLAN
	if err == nil && handled && ok {
		vlan = sender.notificationWireVLAN(packet.VLAN)
		mac = sender.hostNeighborMAC(vlan, route.target)
	}
	s.reloadMu.RUnlock()
	if err != nil {
		return nil, err
	}
	if !handled {
		return packet, nil
	}
	if !ok {
		return nil, errHostEgressUnavailable
	}
	if mac == nil {
		return nil, sender.resolveHostNeighbor(packet, route, vlan)
	}
	resolved := packet.Clone()
	resolved.PutDestMAC(mac)
	return resolved, nil
}

func (s *stackDatagramSender) hostNeighborMAC(vlan int, target netip.Addr) net.HardwareAddr {
	device := s.notificationTargetDevice(vlan, target)
	if device != nil {
		if s.stack.fabric == nil {
			return device.MACAddress
		}
		endpoint, found := s.stack.fabric.interfacesByAddr[target]
		if found && endpoint.device == device && endpoint.network == s.stack.fabric.attachmentNetwork &&
			s.stack.conflictInterfaceAvailable(device, endpoint.interfaceName) {
			return endpoint.mac
		}
	}
	return s.learnedNeighborMAC(vlan, target)
}

func (s *stackDatagramSender) resolveHostNeighbor(packet *Packet, route hostEgressRoute, vlan int) error {
	s.stack.lifecycleMu.RLock()
	if s.stack.stopped {
		s.stack.lifecycleMu.RUnlock()
		return ErrStackStopped
	}
	key := newNotificationNeighborKey(vlan, route.target)
	s.mu.Lock()
	resolution := s.pending[key]
	if s.neighborQueueFull(resolution) {
		s.mu.Unlock()
		s.stack.lifecycleMu.RUnlock()
		return errHostNeighborQueueFull
	}
	first := resolution == nil
	if first {
		resolution = &notificationNeighborResolution{}
		s.pending[key] = resolution
	}
	queued := packet.Clone()
	queued.hostEgressInterface = route.iface.Name
	queued.hostEgressNetwork = route.iface.Network
	resolution.hostPackets = append(resolution.hostPackets, queued)
	s.mu.Unlock()
	s.stack.lifecycleMu.RUnlock()
	if first {
		s.probeNeighbor(key)
	}
	return nil
}

func (s *stackDatagramSender) neighborQueueFull(resolution *notificationNeighborResolution) bool {
	if resolution != nil && len(resolution.notifications)+len(resolution.hostPackets) >= maxPendingNeighborDatagrams {
		return true
	}
	count := 0
	for _, pending := range s.pending {
		count += len(pending.notifications) + len(pending.hostPackets)
	}
	return count >= maxPendingNeighborTotal
}
