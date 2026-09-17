package fabric

import (
	"fmt"
	"net"
	"slices"
	"sort"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// compileAttachments resolves every port-scoped attachment to the network its
// ports land a client on, and records the network-scoped form untouched.
//
// Why a port needs resolving at all: an access switch owns only its own
// management SVI, so the network behind a free access port is never readable
// from the switch that carries the port. It is on whichever device owns that
// VLAN's layer-3 interface, reached over the trunk.
func (c *scenarioCompiler) compileAttachments() {
	if c.cfg == nil {
		return
	}
	for i := range c.cfg.Attachments {
		attachment := &c.cfg.Attachments[i]
		field := fmt.Sprintf("attachments[%d]", i)
		if !c.validateAttachmentForm(attachment, field) {
			continue
		}
		if attachment.At == nil {
			continue
		}
		if compiled, ok := c.compileAttachmentPool(attachment, field); ok {
			c.report.Topology.Attachments = append(c.report.Topology.Attachments, compiled)
		}
	}
}

// validateAttachmentForm enforces exactly one of connect: and at:. Accepting
// both would leave two answers to "where does the tester appear" with no rule
// for which wins.
func (c *scenarioCompiler) validateAttachmentForm(
	attachment *config.LogicalAttachment,
	field string,
) bool {
	switch {
	case attachment.Network != "" && attachment.At != nil:
		c.add(CodeAttachmentFormAmbiguous, field,
			"attachment declares both connect and at; set exactly one")
		return false
	case attachment.Network == "" && attachment.At == nil:
		c.add(CodeAttachmentFormAmbiguous, field,
			"attachment declares neither connect nor at; set exactly one")
		return false
	default:
		return true
	}
}

func (c *scenarioCompiler) compileAttachmentPool(
	attachment *config.LogicalAttachment,
	field string,
) (CompiledAttachment, bool) {
	device := c.deviceByName(attachment.At.Device)
	if device == nil {
		c.add(CodeUnknownAttachmentDevice, field+".at.device",
			"attachment names a device the scenario does not declare")
		return CompiledAttachment{}, false
	}
	if len(attachment.At.Ports) == 0 {
		c.add(CodeAttachmentPoolEmpty, field+".at.ports",
			"attachment pool lists no ports")
		return CompiledAttachment{}, false
	}
	compiled := CompiledAttachment{
		Name:   attachment.Name,
		Device: device.Name,
		Ports:  make([]AttachmentPort, 0, len(attachment.At.Ports)),
	}
	seen := make(map[string]struct{}, len(attachment.At.Ports))
	ok := true
	for j, name := range attachment.At.Ports {
		portField := fmt.Sprintf("%s.at.ports[%d]", field, j)
		if _, duplicate := seen[name]; duplicate {
			c.add(CodeDuplicateAttachmentPort, portField, "port is listed twice in the pool")
			ok = false
			continue
		}
		seen[name] = struct{}{}
		port, resolved := c.compileAttachmentPoolPort(device, name, portField)
		if !resolved {
			ok = false
			continue
		}
		compiled.Ports = append(compiled.Ports, port)
	}
	if !ok {
		return CompiledAttachment{}, false
	}
	if !c.resolvePoolNetwork(&compiled, field) {
		return CompiledAttachment{}, false
	}
	compiled.Pins = c.compileAttachmentPins(attachment, compiled, field)
	return compiled, true
}

// resolvePoolNetwork collapses the pool's ports onto the one network the
// binding reports. A pool spanning two networks has no single answer, and the
// runtime derives one DHCP server and one gateway from the binding -- so it is
// refused here rather than resolved arbitrarily. Per-client placement (AP-2)
// is what lifts this.
func (c *scenarioCompiler) resolvePoolNetwork(compiled *CompiledAttachment, field string) bool {
	networks := make([]string, 0, len(compiled.Ports))
	for _, port := range compiled.Ports {
		if !slices.Contains(networks, port.Network) {
			networks = append(networks, port.Network)
		}
	}
	if len(networks) > 1 {
		sort.Strings(networks)
		c.add(CodeAttachmentPoolNetworksDiffer, field+".at.ports",
			fmt.Sprintf("pool ports land on different networks (%v)", networks))
		return false
	}
	compiled.Network = networks[0]
	return true
}

func (c *scenarioCompiler) compileAttachmentPoolPort(
	device *config.Device,
	name, field string,
) (AttachmentPort, bool) {
	iface := interfaceByName(device, name)
	if iface == nil {
		c.add(CodeUnknownAttachmentPort, field,
			"attachment names a port the device does not declare")
		return AttachmentPort{}, false
	}
	if occupiedBy := portOccupation(device, name); occupiedBy != "" {
		c.add(CodeAttachmentPortOccupied, field,
			"port already carries "+occupiedBy+"; a pool port must be free")
		return AttachmentPort{}, false
	}
	vlan, ok := accessVLAN(iface)
	if !ok {
		c.add(CodeAttachmentPortVLANUnresolved, field,
			"port carries no single access VLAN, so the network behind it is undecidable")
		return AttachmentPort{}, false
	}
	network, resolved := c.networkServingVLAN(vlan, c.broadcastDomain(device, vlan))
	switch resolved {
	case vlanNetworkResolved:
		return AttachmentPort{
			Device: device.Name, Interface: name, VLAN: vlan, Network: network,
		}, true
	case vlanNetworkAmbiguous:
		c.add(CodeAttachmentPortNetworkAmbiguous, field, fmt.Sprintf(
			"VLAN %d is served by more than one network reachable from %s",
			vlan, device.Name,
		))
	case vlanNetworkMissing:
		c.add(CodeAttachmentPortNetworkUnresolved, field, fmt.Sprintf(
			"no network reachable from %s serves VLAN %d", device.Name, vlan,
		))
	}
	return AttachmentPort{}, false
}

func (c *scenarioCompiler) compileAttachmentPins(
	attachment *config.LogicalAttachment,
	compiled CompiledAttachment,
	field string,
) []AttachmentPin {
	pins := make([]AttachmentPin, 0, len(attachment.Pins))
	pinned := make(map[string]struct{}, len(attachment.Pins))
	for j := range attachment.Pins {
		pin := attachment.Pins[j]
		pinField := fmt.Sprintf("%s.pins[%d]", field, j)
		hardware, err := net.ParseMAC(pin.MAC)
		if err != nil {
			c.add(CodeInvalidAttachmentPinMAC, pinField+".mac", "pin MAC is not a MAC address")
			continue
		}
		if !slices.ContainsFunc(compiled.Ports, func(port AttachmentPort) bool {
			return port.Device == pin.Device && port.Interface == pin.Interface
		}) {
			c.add(CodeAttachmentPinOutsidePool, pinField,
				"pin names a port outside this attachment's pool")
			continue
		}
		key := pin.Device + "\x00" + pin.Interface
		if _, taken := pinned[key]; taken {
			c.add(CodeAttachmentPinDuplicate, pinField,
				"two pins claim the same port; a port holds one pinned client")
			continue
		}
		pinned[key] = struct{}{}
		pins = append(pins, AttachmentPin{
			MAC: hardware.String(), Device: pin.Device, Interface: pin.Interface,
		})
	}
	if len(pins) == 0 {
		return nil
	}
	return pins
}

type vlanNetworkResolution int

const (
	vlanNetworkMissing vlanNetworkResolution = iota
	vlanNetworkResolved
	vlanNetworkAmbiguous
)

// networkServingVLAN finds the network a VLAN carries inside that VLAN's
// broadcast domain, by looking for the layer-3 interface that serves it.
//
// The VLAN id alone cannot answer this: measured over the seven generated
// packs, every one reuses its VLAN ids across sites -- a four-site campus has
// four different networks on VLAN 210 -- so a lookup keyed on virtual_vlan
// across the whole scenario is ambiguous by construction. The domain is what
// makes it decidable, and it has to be the domain *of this VLAN*: a domain
// that crossed routed links was still the whole scenario, which is the defect
// AP-3 hit. Every pack reuses VLAN 200 on lab-transit as well, so even a
// single-site pack was ambiguous.
func (c *scenarioCompiler) networkServingVLAN(
	vlan uint16,
	domain map[string]struct{},
) (string, vlanNetworkResolution) {
	var found []string
	for i := range c.cfg.Devices {
		device := &c.cfg.Devices[i]
		if _, reachable := domain[device.Name]; !reachable {
			continue
		}
		for j := range device.Interfaces {
			name := device.Interfaces[j].Network
			if name == "" || slices.Contains(found, name) {
				continue
			}
			if network, exists := c.networks[name]; exists && network.VirtualVLAN == vlan {
				found = append(found, name)
			}
		}
	}
	switch len(found) {
	case 0:
		return "", vlanNetworkMissing
	case 1:
		return found[0], vlanNetworkResolved
	default:
		return "", vlanNetworkAmbiguous
	}
}

// broadcastDomain is every device that one VLAN reaches from the starting
// device, itself included. It walks trunk ports because that is how an edge is
// authored -- there is no separate links section.
//
// Only a link that carries the VLAN extends its domain. Two kinds of link are
// therefore not followed, and both matter:
//
//   - fdb_only ports, which record a learned endpoint rather than an
//     infrastructure link;
//   - any link that does not carry this VLAN -- in particular a routed one,
//     which carries no layer-2 state at all (no VLANs, no native VLAN). A
//     router terminates layer 2; following its links merged every site of a
//     multi-site pack into one "domain", and with VLAN ids reused per site the
//     network behind a free access port was then ambiguous by construction.
func (c *scenarioCompiler) broadcastDomain(
	from *config.Device,
	vlan uint16,
) map[string]struct{} {
	domain := map[string]struct{}{from.Name: {}}
	queue := []string{from.Name}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		device := c.deviceByName(current)
		if device == nil {
			continue
		}
		for _, trunk := range device.TrunkPorts {
			neighbour := trunk.RemoteDevice
			if neighbour == "" || trunk.FDBOnly || !trunkCarriesVLAN(trunk, vlan) {
				continue
			}
			if _, seen := domain[neighbour]; seen {
				continue
			}
			domain[neighbour] = struct{}{}
			queue = append(queue, neighbour)
		}
	}
	return domain
}

// trunkCarriesVLAN reports whether a link forwards one VLAN, tagged in its
// allowed list or untagged as its native VLAN. An empty allowed list is not
// "every VLAN" here: config.IsRoutedTopologyLink reads the same shape as a
// routed link, and the qbridge MIB already builds a device's VLAN list that
// way, so the two agree.
func trunkCarriesVLAN(trunk config.TrunkPort, vlan uint16) bool {
	// Compared as int: the authored fields are int and the caller's VLAN came
	// from accessVLAN, which has already bounded it to 1..4094.
	want := int(vlan)
	if trunk.NativeVLAN == want {
		return true
	}
	return slices.Contains(trunk.VLANs, want)
}

func (c *scenarioCompiler) deviceByName(name string) *config.Device {
	for i := range c.cfg.Devices {
		if c.cfg.Devices[i].Name == name {
			return &c.cfg.Devices[i]
		}
	}
	return nil
}

func interfaceByName(device *config.Device, name string) *config.Interface {
	for i := range device.Interfaces {
		if device.Interfaces[i].Name == name {
			return &device.Interfaces[i]
		}
	}
	return nil
}

// portOccupation names what already uses a port, or "" when it is free. A
// trunk port entry means the port carries a link or a learned client; a
// port-channel membership means it is bundled into a LAG.
func portOccupation(device *config.Device, name string) string {
	for _, trunk := range device.TrunkPorts {
		if trunk.Interface != name {
			continue
		}
		if trunk.FDBOnly {
			return "a client learned in the forwarding database"
		}
		return "a link to " + trunk.RemoteDevice
	}
	for _, channel := range device.PortChannels {
		if slices.Contains(channel.Members, name) {
			return fmt.Sprintf("port-channel%d", channel.ID)
		}
	}
	return ""
}

// accessVLAN is the one VLAN an access port carries. A port with several is a
// trunk, and a port with none has nothing to resolve a network from.
func accessVLAN(iface *config.Interface) (uint16, bool) {
	if len(iface.VLANs) != 1 {
		return 0, false
	}
	vlan := iface.VLANs[0]
	if vlan < 1 || vlan > 4094 {
		return 0, false
	}
	return uint16(vlan), true
}
