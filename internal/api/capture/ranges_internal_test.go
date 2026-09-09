package capture

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/packetdecode"
)

func TestRangeMemoryAccounting(t *testing.T) {
	result := &AnalysisResult{Packets: []Packet{{}}}
	before := estimateResultSize(result)
	result.Packets[0].ByteRanges = []packetdecode.ByteRange{{Layer: "tcp", Field: "Source Port", Start: 38, End: 40}}
	after := estimateResultSize(result)
	if after <= before {
		t.Fatalf("range storage not counted: before=%d after=%d", before, after)
	}
}
