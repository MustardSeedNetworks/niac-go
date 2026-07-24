package api

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestBasicMetricsIncludesUDPProxyOverloadDrops(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	writeBasicMetrics(&output, &protocols.Statistics{UDPProxyOverloadDrops: 3}, 0)

	if !strings.Contains(output.String(), "niac_udp_proxy_overload_drops_total 3\n") {
		t.Fatalf("metrics missing UDP proxy overload counter:\n%s", output.String())
	}
}
