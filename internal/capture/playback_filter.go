package capture

import (
	"fmt"

	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"golang.org/x/net/bpf"
)

// bpfSnaplen is the capture length used when compiling a replay BPF filter; it
// bounds how far into each packet the program may read. Full frames (including
// jumbo) fit within it.
const bpfSnaplen = 262144

// ValidateBPFExpr reports whether expr is a compilable BPF filter. It compiles
// against Ethernet — the common replay link type — purely as a syntax gate so
// the API can reject a bad expression up front; playback recompiles against the
// capture's actual link type at Start.
func ValidateBPFExpr(expr string) error {
	if _, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, bpfSnaplen, expr); err != nil {
		return fmt.Errorf("invalid BPF filter: %w", err)
	}

	return nil
}

// compileBPFVM compiles expr for linkType into a pure-Go VM that matches packet
// bytes. Compilation uses libpcap (already linked for live capture) but no live
// handle — so the replay read path stays CGO-free and the VM runs in Go.
func compileBPFVM(linkType layers.LinkType, expr string) (*bpf.VM, error) {
	insns, err := pcap.CompileBPFFilter(linkType, bpfSnaplen, expr)
	if err != nil {
		return nil, fmt.Errorf("compile BPF filter: %w", err)
	}

	program := make([]bpf.Instruction, len(insns))
	for i, in := range insns {
		program[i] = bpf.RawInstruction{Op: in.Code, Jt: in.Jt, Jf: in.Jf, K: in.K}.Disassemble()
	}

	vm, err := bpf.NewVM(program)
	if err != nil {
		return nil, fmt.Errorf("build BPF program: %w", err)
	}

	return vm, nil
}

// matchesFilter reports whether data passes the compiled filter. A nil VM (no
// filter configured) passes every packet.
func (p *PlaybackEngine) matchesFilter(data []byte) bool {
	if p.bpfVM == nil {
		return true
	}

	n, err := p.bpfVM.Run(data)
	if err != nil {
		// A compiled program shouldn't fail on well-formed input; drop the
		// packet rather than send something the filter couldn't vet.
		p.logError("BPF filter evaluation failed", "error", err)

		return false
	}

	return n > 0
}
