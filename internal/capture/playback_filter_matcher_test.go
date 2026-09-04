package capture

import "testing"

// arpFrame returns a minimal Ethernet frame carrying EtherType 0x0806.
func arpFrame() []byte {
	frame := make([]byte, 60)
	copy(frame, []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0x02, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06,
	})

	return frame
}

func TestNewEthernetMatcherWithNoExpressionPassesEverything(t *testing.T) {
	match, err := NewEthernetMatcher("")
	if err != nil {
		t.Fatalf("NewEthernetMatcher: %v", err)
	}
	if !match(arpFrame()) {
		t.Error("empty filter dropped a frame")
	}
}

func TestNewEthernetMatcherSelectsByProtocol(t *testing.T) {
	match, err := NewEthernetMatcher("arp")
	if err != nil {
		t.Fatalf("NewEthernetMatcher: %v", err)
	}
	if !match(arpFrame()) {
		t.Error("`arp` dropped an ARP frame")
	}

	ip := arpFrame()
	ip[12], ip[13] = 0x08, 0x00
	if match(ip) {
		t.Error("`arp` matched an IPv4 frame")
	}
}

func TestNewEthernetMatcherRejectsAnUncompilableExpression(t *testing.T) {
	if _, err := NewEthernetMatcher("nonsense &&"); err == nil {
		t.Error("expected a compile error")
	}
}
