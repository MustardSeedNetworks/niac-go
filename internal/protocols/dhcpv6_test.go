package protocols

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// buildDHCPv6WithClientID assembles a minimal DHCPv6 Solicit carrying a
// Client-ID option whose DUID is duidLen bytes.
func buildDHCPv6WithClientID(duidLen int) []byte {
	msg := []byte{1, 0x00, 0x00, 0x01} // Solicit + transaction ID
	opt := make([]byte, 4+duidLen)
	binary.BigEndian.PutUint16(opt[0:2], DHCPv6OptClientID)
	binary.BigEndian.PutUint16(opt[2:4], uint16(duidLen))

	return append(msg, opt...)
}

// TestParseDHCPv6RejectsOversizedDUID verifies the parser discards a message
// whose Client-ID DUID exceeds the RFC 8415 limit. Accepting it would echo the
// oversized DUID into the reply and overflow the MTU (a dropped response).
func TestParseDHCPv6RejectsOversizedDUID(t *testing.T) {
	h := &DHCPv6Handler{}

	// A conformant DUID (DUID-LL, 10 bytes) must parse.
	if _, err := h.parseDHCPv6Message(buildDHCPv6WithClientID(duidLLSize)); err != nil {
		t.Fatalf("valid Client-ID rejected: %v", err)
	}

	// One octet over the cap must be rejected as malformed.
	if _, err := h.parseDHCPv6Message(buildDHCPv6WithClientID(maxDUIDSize + 1)); !errors.Is(err, ErrDUIDTooLong) {
		t.Errorf("oversized DUID: got err %v, want ErrDUIDTooLong", err)
	}

	// The boundary value itself is still accepted.
	if _, err := h.parseDHCPv6Message(buildDHCPv6WithClientID(maxDUIDSize)); err != nil {
		t.Errorf("DUID at the cap rejected: %v", err)
	}
}

// TestParseDHCPv6PreservesClientID sanity-checks that a normal Client-ID is
// parsed intact (regression guard for the length check above).
func TestParseDHCPv6PreservesClientID(t *testing.T) {
	h := &DHCPv6Handler{}
	raw := buildDHCPv6WithClientID(duidLLSize)

	msg, err := h.parseDHCPv6Message(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	got := h.findOption(msg, DHCPv6OptClientID)
	if got == nil {
		t.Fatal("Client-ID option missing after parse")
	}
	if !bytes.Equal(got.Data, raw[8:]) {
		t.Errorf("Client-ID data = %x, want %x", got.Data, raw[8:])
	}
}
