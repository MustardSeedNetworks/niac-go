package protocols

import (
	"testing"
)

func TestICMPv6TypeNames(t *testing.T) {
	handler := &ICMPv6Handler{debugLevel: 0}

	tests := []struct {
		msgType  uint8
		expected string
	}{
		{ICMPv6TypeEchoRequest, "Echo Request"},
		{ICMPv6TypeEchoReply, "Echo Reply"},
		{ICMPv6TypeNeighborSolicitation, "Neighbor Solicitation"},
		{ICMPv6TypeNeighborAdvertisement, "Neighbor Advertisement"},
		{ICMPv6TypeRouterSolicitation, "Router Solicitation"},
		{ICMPv6TypeRouterAdvertisement, "Router Advertisement"},
		{ICMPv6TypeDestUnreachable, "Destination Unreachable"},
		{ICMPv6TypePacketTooBig, "Packet Too Big"},
		{ICMPv6TypeTimeExceeded, "Time Exceeded"},
		{ICMPv6TypeParameterProblem, "Parameter Problem"},
		{ICMPv6TypeRedirect, "Redirect"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := handler.getTypeName(tt.msgType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestICMPv6Constants(t *testing.T) {
	// Verify ICMPv6 type values match RFC 4443
	if ICMPv6TypeDestUnreachable != 1 {
		t.Error("ICMPv6TypeDestUnreachable should be 1")
	}
	if ICMPv6TypePacketTooBig != 2 {
		t.Error("ICMPv6TypePacketTooBig should be 2")
	}
	if ICMPv6TypeTimeExceeded != 3 {
		t.Error("ICMPv6TypeTimeExceeded should be 3")
	}
	if ICMPv6TypeParameterProblem != 4 {
		t.Error("ICMPv6TypeParameterProblem should be 4")
	}
	if ICMPv6TypeEchoRequest != 128 {
		t.Error("ICMPv6TypeEchoRequest should be 128")
	}
	if ICMPv6TypeEchoReply != 129 {
		t.Error("ICMPv6TypeEchoReply should be 129")
	}

	// Verify NDP message types (RFC 4861)
	if ICMPv6TypeRouterSolicitation != 133 {
		t.Error("ICMPv6TypeRouterSolicitation should be 133")
	}
	if ICMPv6TypeRouterAdvertisement != 134 {
		t.Error("ICMPv6TypeRouterAdvertisement should be 134")
	}
	if ICMPv6TypeNeighborSolicitation != 135 {
		t.Error("ICMPv6TypeNeighborSolicitation should be 135")
	}
	if ICMPv6TypeNeighborAdvertisement != 136 {
		t.Error("ICMPv6TypeNeighborAdvertisement should be 136")
	}
	if ICMPv6TypeRedirect != 137 {
		t.Error("ICMPv6TypeRedirect should be 137")
	}
}

func TestICMPv6OptionConstants(t *testing.T) {
	// Verify option type values match RFC 4861
	if ICMPv6OptSourceLinkAddr != 1 {
		t.Error("ICMPv6OptSourceLinkAddr should be 1")
	}
	if ICMPv6OptTargetLinkAddr != 2 {
		t.Error("ICMPv6OptTargetLinkAddr should be 2")
	}
	if ICMPv6OptPrefixInfo != 3 {
		t.Error("ICMPv6OptPrefixInfo should be 3")
	}
	if ICMPv6OptRedirectedHdr != 4 {
		t.Error("ICMPv6OptRedirectedHdr should be 4")
	}
	if ICMPv6OptMTU != 5 {
		t.Error("ICMPv6OptMTU should be 5")
	}
}

func TestNDFlags(t *testing.T) {
	// Verify NDP flag values
	if NDFlagRouter != 0x80 {
		t.Error("NDFlagRouter should be 0x80")
	}
	if NDFlagSolicited != 0x40 {
		t.Error("NDFlagSolicited should be 0x40")
	}
	if NDFlagOverride != 0x20 {
		t.Error("NDFlagOverride should be 0x20")
	}

	// Test flag combinations
	flags := NDFlagSolicited | NDFlagOverride
	if flags != 0x60 {
		t.Errorf("Solicited+Override flags should be 0x60, got 0x%02x", flags)
	}
}
