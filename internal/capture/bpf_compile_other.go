//go:build !darwin

package capture

import (
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

func compileNativeBPFInstructions(linkType layers.LinkType, expr string) ([]pcap.BPFInstruction, error) {
	return pcap.CompileBPFFilter(linkType, bpfSnaplen, expr)
}
