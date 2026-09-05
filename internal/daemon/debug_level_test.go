package daemon

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// errStopHere ends the start once the seam has reported the level, so the test
// never touches an interface.
var errStopHere = errors.New("stop after recording the debug level")

func debugLevelRequest() api.SimulationRequest {
	return api.SimulationRequest{
		Interface: "missing-debug-level-interface",
		ConfigData: `devices:
  - name: debug-level-router
    type: router
    mac: "02:00:00:00:00:01"
    ips:
      - "192.0.2.10"
`,
	}
}

// A `--once` run has no listener, so PUT /api/v1/debug/level cannot reach it:
// Config.DebugLevel is the only verbosity control it has, and it has to arrive
// at the stack the run actually starts.
func TestConfigDebugLevelReachesTheSimulation(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "1")

	for _, want := range []int{0, 3} {
		d, err := NewDaemon(Config{StoragePath: "disabled", DebugLevel: want})
		if err != nil {
			t.Fatalf("NewDaemon: %v", err)
		}
		d.apiServer = api.NewServer(api.ServerConfig{})

		got := -1
		d.startSimulation = func(
			_ string,
			_ *config.Config,
			_ *fabric.Topology,
			_ bool,
			debugLevel int,
		) (simulationResources, error) {
			got = debugLevel

			return simulationResources{cancel: func() {}}, errStopHere
		}

		_ = d.StartSimulation(debugLevelRequest())
		if got != want {
			t.Errorf("simulation started at debug level %d, want %d", got, want)
		}
	}
}
