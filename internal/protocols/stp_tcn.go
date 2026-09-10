package protocols

import (
	"net"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// SendTopologyChange emits the short IEEE 802.1D TCN on the device's segment.
func (h *STPHandler) SendTopologyChange(device *config.Device) error {
	if len(device.MACAddress) != SizeOfMac {
		return ErrDeviceNoMACAddress
	}
	const minimumFrameLength = 60 // Ethernet frame excluding the hardware FCS.
	frame := make([]byte, minimumFrameLength)
	destination, err := net.ParseMAC(STPMulticastMAC)
	if err != nil {
		return err
	}
	copy(frame, destination)
	copy(frame[SizeOfMac:], device.MACAddress)
	copy(
		frame[ethernetHeaderSize-2:],
		[]byte{0, 7, stpLLCDSAP, stpLLCSSAP, stpLLCControl, 0, 0, STPVersion, BPDUTypeTCN},
	)
	return h.stack.send(
		&Packet{Buffer: frame, Length: len(frame), Device: device, VLAN: h.stack.discoveryVLAN(device)},
	)
}
