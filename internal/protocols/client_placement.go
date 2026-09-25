package protocols

import (
	"net"
	"sync"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// clientPlacement decides which pool port each client MAC is plugged into.
//
// A pinned MAC always lands on its pin, and a pinned port is never handed to
// anyone else, so a pin holds even when its client is the last to arrive. Every
// other MAC takes the first free unpinned port in the pool's authored order, in
// the order the clients were first seen. A placement is sticky for the session:
// a client that falls silent keeps its port, as a tester left plugged in does.
type clientPlacement struct {
	mu sync.Mutex
	// device carries the whole pool: an attachment names one switch.
	device   string
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
		device:   attachment.Device,
		earliest: -1,
		ports:    append([]fabric.AttachmentPort(nil), attachment.Ports...),
		pins:     make(map[string]int, len(attachment.Pins)),
		reserved: make([]bool, len(attachment.Ports)),
		taken:    make([]bool, len(attachment.Ports)),
		assigned: make(map[string]int),
	}
	for _, pin := range attachment.Pins {
		for index, port := range placement.ports {
			if port.Device == pin.Device && port.Interface == pin.Interface {
				placement.pins[pin.MAC] = index
				placement.reserved[index] = true
				break
			}
		}
	}

	return placement
}

// assign gives mac its port on first sight and returns it. It reports false
// when mac already has a port, which is every frame after its first, or when
// the pool has none left for it.
func (p *clientPlacement) assign(mac string) (fabric.AttachmentPort, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.assigned[mac]; ok {
		return fabric.AttachmentPort{}, false
	}

	index, ok := p.pins[mac]
	if !ok {
		index = p.firstFreePort()
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

func (p *clientPlacement) firstFreePort() int {
	for index := range p.ports {
		if !p.reserved[index] && !p.taken[index] {
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

// advertisedPort is the port the pool's switch names in its discovery
// advertisements. A switch sends one per port, but every client here shares
// one wire and hears every frame, so one advertisement has to serve them all:
// it names the earliest-placed client's port and, before anyone is placed,
// the port the next unpinned client will take, which is where a passive
// listener lands once it transmits.
func (p *clientPlacement) advertisedPort() fabric.AttachmentPort {
	p.mu.Lock()
	defer p.mu.Unlock()

	index := p.earliest
	if index < 0 {
		index = max(p.firstFreePort(), 0)
	}

	return p.ports[index]
}

func (p *clientPlacement) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	clear(p.taken)
	clear(p.assigned)
	p.earliest = -1
	p.exhausted = false
}

// placeObservedClient plugs a client into its pool port on first sight: the
// port's device learns the MAC there, and no other device learns it at all.
func (s *Stack) placeObservedClient(mac net.HardwareAddr) {
	if s.fabric == nil || s.fabric.placement == nil {
		return
	}

	port, assigned := s.fabric.placement.assign(mac.String())
	if !assigned {
		return
	}

	device := s.fabric.devicesByName[port.Device]
	if device == nil {
		return
	}
	if !s.snmpAgents[device].placeLearnedClient(mac, port.Interface, int(port.VLAN)) {
		logging.Debugf("Attachment: %s on %s %s has no bridge row to learn it on",
			mac, port.Device, port.Interface)
	}
}
