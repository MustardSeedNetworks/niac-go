package packetdecode_test

import (
	"encoding/hex"
	"encoding/json"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/packetdecode"
)

func TestDecodedByteRanges(t *testing.T) {
	tests := []struct {
		name, frame, layer, field string
		start, end                int
	}{
		{
			"tagged TCP",
			"00112233445566778899aabb810000c80800450000280000000040060000c0000201c6336402005001bb00000001000000005002200000000000",
			"tcp",
			"Source Port",
			38,
			40,
		},
		{
			"IPv4 options",
			"00112233445566778899aabb08004600002c0000000040060000c0000201c633640201010101005001bb00000001000000005002200000000000",
			"tcp",
			"Source Port",
			38,
			40,
		},
		{
			"TCP options",
			"00112233445566778899aabb08004500002c0000000040060000c0000201c6336402005001bb00000001000000006002200000000000020405b4",
			"tcp",
			"Options",
			54,
			58,
		},
		{
			"IPv6 extension",
			"00112233445566778899aabb86dd60000000001c004020010db800000000000000000000000120010db80000000000000000000000020600000000000000005001bb00000001000000005002200000000000",
			"tcp",
			"Source Port",
			62,
			64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tt.frame)
			if err != nil {
				t.Fatal(err)
			}
			out := map[string]any{}
			packetdecode.Enrich(out, raw)
			ranges, ok := out["byte_ranges"].([]packetdecode.ByteRange)
			if !ok {
				t.Fatalf("missing ranges: %v", out)
			}
			want := packetdecode.ByteRange{Layer: tt.layer, Field: tt.field, Start: tt.start, End: tt.end}
			assertByteRange(t, ranges, want)
		})
	}
}

func TestRangeJSONForMalformedFrame(t *testing.T) {
	out := map[string]any{}
	packetdecode.Enrich(out, []byte{1, 2, 3})
	raw, err := json.Marshal(out["byte_ranges"])
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "[]" {
		t.Fatalf("malformed frame ranges = %s; want []", raw)
	}
}

func TestRangesMatchFirstVLANOccurrence(t *testing.T) {
	raw, err := hex.DecodeString(
		"00112233445566778899aabb810000c88100012c0800450000280000000040060000c0000201c6336402005001bb00000001000000005002200000000000",
	)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	packetdecode.Enrich(out, raw)
	ranges, ok := out["byte_ranges"].([]packetdecode.ByteRange)
	if !ok {
		t.Fatal("ranges missing")
	}
	count := 0
	for _, span := range ranges {
		if span.Layer == "dot1q" && span.Field == "VLAN" {
			count++
			if span.Start != 14 {
				t.Errorf("displayed outer VLAN highlights %d", span.Start)
			}
		}
	}
	if count != 1 {
		t.Errorf("VLAN ranges = %d; want one matching displayed header", count)
	}
}

func assertByteRange(t *testing.T, ranges []packetdecode.ByteRange, want packetdecode.ByteRange) {
	t.Helper()
	if slices.Contains(ranges, want) {
		return
	}
	t.Fatalf("missing %+v in %v", want, ranges)
}
