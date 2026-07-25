package protocols

import "net"

type discoveryCapture struct {
	sources []string
}

func (*discoveryCapture) ReadPacket([]byte) ([]byte, error) { return nil, nil }

func (c *discoveryCapture) SendPacket(frame []byte) error {
	c.sources = append(c.sources, net.HardwareAddr(frame[6:12]).String())
	return nil
}

func (*discoveryCapture) SetFilter(string) error { return nil }
func (*discoveryCapture) Filter() string         { return "" }
