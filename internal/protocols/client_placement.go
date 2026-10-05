package protocols

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// ErrAttachmentPortOccupied refuses a re-pin onto a port another client is
// plugged into.
var ErrAttachmentPortOccupied = errors.New("another client is plugged into that port")

// errAttachmentPoolNotBound refuses a re-pin on a session whose binding does
// not land on a port pool.
var errAttachmentPoolNotBound = errors.New("the session is not bound to an attachment pool")

// errAttachmentPinMissing refuses a re-pin whose recompiled pool has no pin
// for the client being moved.
var errAttachmentPinMissing = errors.New("the attachment pool has no pin for this client")

// ErrAttachmentPortShut refuses a re-pin onto an administratively shut port: a
// cable plugged into it gets no link, so the client would be cut off.
var ErrAttachmentPortShut = errors.New("that port is administratively shut")

// portUsable reports whether a pool port can carry a client. A shut port
// cannot, whatever is plugged into it.
type portUsable func(fabric.AttachmentPort) bool

// clientPlacement decides which pool port each client MAC is plugged into.
//
// A pinned MAC always lands on its pin, and a pinned port is never handed to
// anyone else, so a pin holds even when its client is the last to arrive. Every
// other MAC takes the first free unpinned port in the pool's authored order, in
// the order the clients were first seen. A shut port takes no one, pinned or
// not. A placement is sticky for the session: a client that falls silent keeps
// its port, as a tester left plugged in does.
type clientPlacement struct {
	mu       sync.Mutex
	ports    []fabric.AttachmentPort
	pins     map[string]int
	reserved []bool
	taken    []bool
	assigned map[string]int
	// earliest is the port of the first client placed, -1 until there is one.
	earliest int
	// exhausted is set once a client found no free port, so the log says so
	// once per session rather than on every frame that client sends.
	exhausted bool
}

func newClientPlacement(attachment fabric.CompiledAttachment) *clientPlacement {
	placement := &clientPlacement{
		earliest: -1,
		ports:    append([]fabric.AttachmentPort(nil), attachment.Ports...),
		taken:    make([]bool, len(attachment.Ports)),
		assigned: make(map[string]int),
	}
	placement.pins, placement.reserved = placement.resolvePins(attachment.Pins)

	return placement
}

func (p *clientPlacement) resolvePins(pins []fabric.AttachmentPin) (map[string]int, []bool) {
	indexes := make(map[string]int, len(pins))
	reserved := make([]bool, len(p.ports))
	for _, pin := range pins {
		for index, port := range p.ports {
			if port.Device == pin.Device && port.Interface == pin.Interface {
				indexes[pin.MAC] = index
				reserved[index] = true
				break
			}
		}
	}

	return indexes, reserved
}

// clientMove is where a re-pinned client was plugged in and where it is now.
// A client not yet seen has no port to leave, so from is unset.
type clientMove struct {
	from, to fabric.AttachmentPort
	placed   bool
}

// repin replaces the pool's pins with the recompiled set, which fixes mac to
// a new port, and moves mac there if it is already plugged in. Every other
// client stays where it is: a port another client holds is refused rather
// than taken from it, as a real port with a cable in it would be.
func (p *clientPlacement) repin(pins []fabric.AttachmentPin, mac string, usable portUsable) (clientMove, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	indexes, reserved := p.resolvePins(pins)
	target, ok := indexes[mac]
	if !ok {
		return clientMove{}, fmt.Errorf("%w: %s", errAttachmentPinMissing, mac)
	}
	current, placed := p.assigned[mac]
	if p.taken[target] && (!placed || current != target) {
		return clientMove{}, fmt.Errorf("%w: %s %s",
			ErrAttachmentPortOccupied, p.ports[target].Device, p.ports[target].Interface)
	}
	if !usable(p.ports[target]) {
		return clientMove{}, fmt.Errorf("%w: %s %s",
			ErrAttachmentPortShut, p.ports[target].Device, p.ports[target].Interface)
	}

	p.pins, p.reserved = indexes, reserved
	if !placed {
		return clientMove{to: p.ports[target]}, nil
	}
	p.taken[current] = false
	p.taken[target] = true
	p.assigned[mac] = target
	if p.earliest == current {
		p.earliest = target
	}

	return clientMove{from: p.ports[current], to: p.ports[target], placed: true}, nil
}

// assign gives mac its port on first sight and returns it. It reports false
// when mac already has a port, which is every frame after its first, when its
// pinned port is shut, or when the pool has no usable port left for it.
func (p *clientPlacement) assign(mac string, usable portUsable) (fabric.AttachmentPort, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.assigned[mac]; ok {
		return fabric.AttachmentPort{}, false
	}

	index, ok := p.pins[mac]
	if ok && !usable(p.ports[index]) {
		return fabric.AttachmentPort{}, false
	}
	if !ok {
		index = p.firstFreePort(usable)
	}
	if index < 0 {
		if !p.exhausted {
			p.exhausted = true
			logging.Infof("Attachment pool full: client %s has no free port (%d ports)", mac, len(p.ports))
		}
		return fabric.AttachmentPort{}, false
	}

	p.taken[index] = true
	p.assigned[mac] = index
	if p.earliest < 0 {
		p.earliest = index
	}

	return p.ports[index], true
}

func (p *clientPlacement) firstFreePort(usable portUsable) int {
	for index := range p.ports {
		if !p.reserved[index] && !p.taken[index] && usable(p.ports[index]) {
			return index
		}
	}

	return -1
}

func (p *clientPlacement) lookup(mac string) (fabric.AttachmentPort, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	index, ok := p.assigned[mac]
	if !ok {
		return fabric.AttachmentPort{}, false
	}

	return p.ports[index], true
}

// advertisedPort is the pool port named in discovery advertisements, by the
// switch that carries it. A switch sends one per port, but every client here
// shares one wire and hears every frame, so one advertisement has to serve
// them all: it names the earliest-placed client's port and, before anyone is
// placed, the port the next unpinned client will take, which is where a
// passive listener lands once it transmits. A shut port advertises nothing, so once
// the earliest client's port is shut the next free port speaks instead, and
// with no usable port left the switch is silent.
func (p *clientPlacement) advertisedPort(usable portUsable) (fabric.AttachmentPort, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	index := p.earliest
	if index < 0 || !usable(p.ports[index]) {
		index = p.firstFreePort(usable)
	}
	if index < 0 {
		return fabric.AttachmentPort{}, false
	}

	return p.ports[index], true
}

// reset unplugs every client and returns the ports they were plugged into.
func (p *clientPlacement) reset() []fabric.AttachmentPort {
	p.mu.Lock()
	defer p.mu.Unlock()

	var plugged []fabric.AttachmentPort
	for index, taken := range p.taken {
		if taken {
			plugged = append(plugged, p.ports[index])
		}
	}
	clear(p.taken)
	clear(p.assigned)
	p.earliest = -1
	p.exhausted = false

	return plugged
}

// placeObservedClient plugs a client into its pool port on first sight: the
// port comes up, its device learns the MAC there, and no other device learns
// it at all.
func (s *Stack) placeObservedClient(mac net.HardwareAddr) {
	if s.fabric == nil || s.fabric.placement == nil {
		return
	}

	port, assigned := s.fabric.placement.assign(mac.String(), s.fabric.portAdminUp)
	if !assigned {
		return
	}

	device := s.fabric.devicesByName[port.Device]
	if device == nil {
		return
	}
	s.setPoolPortCable(device, port.Interface, true)
	if !s.snmpAgents[device].placeLearnedClient(mac, port.Interface, int(port.VLAN)) {
		logging.Debugf("Attachment: %s on %s %s has no bridge row to learn it on",
			mac, port.Device, port.Interface)
	}
}

// RepinAttachedClient moves one client to its new pin on the running session.
// topology is the session's scenario recompiled with that pin, so the compile
// has already refused a pin outside the pool or on another client's pin.
//
// Nothing is rebuilt: a reload would reset every DHCP handler, SNMP agent and
// placement, and so unplug every other client along with this one. Only the
// pool's pins and the forwarding entries for mac change, and the client keeps
// its lease.
func (s *Stack) RepinAttachedClient(topology *fabric.Topology, mac net.HardwareAddr) error {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	if s.fabric == nil || s.fabric.placement == nil {
		return errAttachmentPoolNotBound
	}
	index := slices.IndexFunc(topology.Attachments, func(attachment fabric.CompiledAttachment) bool {
		return attachment.Name == s.fabric.binding.Attachment
	})
	if index < 0 {
		return errAttachmentPoolNotBound
	}
	move, err := s.fabric.placement.repin(topology.Attachments[index].Pins, mac.String(), s.fabric.portAdminUp)
	if err != nil {
		return err
	}
	s.fabric.topology.Attachments = slices.Clone(topology.Attachments)
	if !move.placed || move.from == move.to {
		return nil
	}

	// Every pool port lands on one network, so the client keeps its lease. On
	// the same switch placing it rewrites its port in the entry keyed by its
	// VLAN; a move to another switch also takes it out of the old switch's
	// forwarding table, as unplugging the cable ages it out there.
	from := s.fabric.devicesByName[move.from.Device]
	to := s.fabric.devicesByName[move.to.Device]
	s.setPoolPortCable(from, move.from.Interface, false)
	s.setPoolPortCable(to, move.to.Interface, true)
	if from != to {
		s.snmpAgents[from].forgetLearnedClient(mac, int(move.from.VLAN))
	}
	if !s.snmpAgents[to].placeLearnedClient(mac, move.to.Interface, int(move.to.VLAN)) {
		logging.Debugf("Attachment: %s on %s %s has no bridge row to learn it on",
			mac, move.to.Device, move.to.Interface)
	}
	logging.Infof("Attachment: moved %s from %s %s to %s %s",
		mac, move.from.Device, move.from.Interface, move.to.Device, move.to.Interface)

	return nil
}

// unplugPoolClients returns every pool port a client was placed on to the link
// state it was authored with, "notconnect" for a spare port, as unplugging the
// tester does on a real switch.
func (s *Stack) unplugPoolClients() {
	for _, port := range s.fabric.placement.reset() {
		if device := s.fabric.devicesByName[port.Device]; device != nil {
			s.setPoolPortCable(device, port.Interface, false)
		}
	}
}

// setPoolPortCable plugs a cable into a pool port or pulls it out. Carrier
// follows the cable and operational state follows carrier and admin state, as
// a "shutdown" on the port leaves it. Pulling the cable restores the carrier
// the port was authored with, so a pool port authored connected stays up.
func (s *Stack) setPoolPortCable(device *config.Device, name string, plugged bool) {
	state := s.deviceStates[device]
	if state == nil {
		return
	}
	carrier := plugged || statusUp(findConfigInterface(device, name).OperStatus)
	err := state.UpdateInterface(name, func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.CarrierUp = carrier
		iface.OperUp = iface.AdminUp && carrier
		return iface, nil
	})
	if err != nil {
		logging.Errorf("Attachment: %s %s link state not updated: %v", device.Name, name, err)
	}
}
