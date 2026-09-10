package protocols

import (
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *stackDatagramSender) sendNotificationIntent(notification pendingNotification, reloadHeld bool) error {
	current, target, mac, err := s.prepareNotificationIntent(notification)
	if err != nil {
		return err
	}
	return s.dispatchNotification(current, target, mac, reloadHeld)
}

func (s *stackDatagramSender) dispatchNotification(
	current pendingNotification, target netip.Addr, mac net.HardwareAddr, reloadHeld bool,
) error {
	if mac == nil {
		return s.queueNotificationNeighbor(current, target, reloadHeld)
	}
	return s.emitNotification(current.egressDevice, current.vlan, current.source, current.destination,
		current.sourcePort, current.destinationPort, mac, current.payload, current.routed)
}

func (s *stackDatagramSender) prepareNotificationIntent(
	notification pendingNotification,
) (pendingNotification, netip.Addr, net.HardwareAddr, error) {
	device := notification.originDevice
	store := s.stack.deviceStates[device]
	if store == nil || len(device.MACAddress) != SizeOfMac {
		return notification, netip.Addr{}, nil, errHostEgressUnavailable
	}
	snapshot := store.Snapshot()
	if notification.originInterface != "" && !s.notificationOriginValid(notification, snapshot) {
		return notification, netip.Addr{}, nil, errHostEgressUnavailable
	}
	snapshot, err := s.projectNotificationMask(notification, snapshot)
	if err != nil {
		return notification, netip.Addr{}, nil, err
	}
	source, target := notificationSnapshotRoute(snapshot, notification.destination)
	if !source.IsValid() {
		return notification, target, nil, errHostEgressUnavailable
	}
	iface := notificationSourceInterface(snapshot.Network.Interfaces, source)
	local := notification.destination.Is4() && hostMaskRole(device) &&
		s.notificationLocalInterface(device, iface) && s.stack.hostPacketVLAN(
		&Packet{generatedHost: device, VLAN: notification.vlan},
	)
	if notification.originInterface != "" && (!local || iface.Name != notification.originInterface) {
		return notification, target, nil, errHostEgressUnavailable
	}
	if local {
		notification.originInterface, notification.originNetwork = iface.Name, iface.Network
	} else {
		notification.originDevice = nil
	}
	notification.source = source
	notification.egressDevice, notification.neighborSource, notification.routed = device, source, false
	if local && notificationFaultedInterface(snapshot, iface.Name) {
		return notification, target, s.hostNeighborMAC(notification.vlan, target), nil
	}
	notification.egressDevice, notification.neighborSource, target, notification.routed = s.notificationEgress(
		device, source, notification.destination, target, notification.vlan,
	)
	mac, err := s.notificationDestinationMAC(
		notification.egressDevice,
		notification.vlan,
		notification.destination,
		target,
	)
	return notification, target, mac, err
}

func (s *stackDatagramSender) projectNotificationMask(
	notification pendingNotification, snapshot devicestate.Snapshot,
) (devicestate.Snapshot, error) {
	device := notification.originDevice
	if !notification.destination.Is4() || !hostMaskRole(device) {
		snapshot.PrefixFaults = nil
		return snapshot, nil
	}
	snapshot.PrefixFaults = s.localNotificationFaults(device, snapshot)
	if len(snapshot.PrefixFaults) == 0 {
		return snapshot, nil
	}
	if !s.stack.hostPacketVLAN(&Packet{generatedHost: device, VLAN: notification.vlan}) {
		return snapshot, errHostEgressUnavailable
	}
	snapshot.Network = snapshot.EffectiveHostNetwork()
	return snapshot, nil
}

func notificationSourceInterface(interfaces []devicestate.Interface, address netip.Addr) devicestate.Interface {
	for _, iface := range interfaces {
		if iface.Address.Addr() == address {
			return iface
		}
	}
	return devicestate.Interface{}
}

func notificationFaultedInterface(snapshot devicestate.Snapshot, name string) bool {
	for _, fault := range snapshot.PrefixFaults {
		if fault.Interface == name {
			return true
		}
	}
	return false
}

func (s *stackDatagramSender) notificationLocalInterface(device *config.Device, iface devicestate.Interface) bool {
	if !devicestate.ValidFaultAddress(iface.Address.Addr()) || !iface.AdminUp || !iface.OperUp {
		return false
	}
	return s.notificationAttachedInterface(device, iface.Name)
}

func (s *stackDatagramSender) notificationAttachedInterface(device *config.Device, name string) bool {
	if s.stack.fabric == nil {
		return true
	}
	for _, endpoint := range s.stack.fabric.interfacesByAddr {
		if endpoint.device == device && endpoint.interfaceName == name &&
			endpoint.network == s.stack.fabric.attachmentNetwork {
			return true
		}
	}
	return false
}

func (s *stackDatagramSender) localNotificationFaults(
	device *config.Device, snapshot devicestate.Snapshot,
) []devicestate.InterfacePrefixFault {
	var faults []devicestate.InterfacePrefixFault
	for _, fault := range snapshot.PrefixFaults {
		if s.notificationAttachedInterface(device, fault.Interface) {
			faults = append(faults, fault)
		}
	}
	return faults
}

func (s *stackDatagramSender) notificationOriginValid(
	notification pendingNotification, snapshot devicestate.Snapshot,
) bool {
	for _, iface := range snapshot.Network.Interfaces {
		if iface.Name == notification.originInterface {
			return iface.Network == notification.originNetwork &&
				s.notificationLocalInterface(notification.originDevice, iface)
		}
	}
	return false
}
