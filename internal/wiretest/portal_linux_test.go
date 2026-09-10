//go:build linux && integration

package wiretest_test

import (
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestCaptivePortalOnTCPWire(t *testing.T) {
	requireWire(t)
	stack := startPortalStack(t)
	baseline := readPortalWire(t, "10.254.200.11/status")
	peer := readPortalWire(t, "10.254.200.12/status")
	if baseline.status != http.StatusCreated || baseline.body != "healthy portal-host" {
		t.Fatalf("unexpected healthy response: %+v", baseline)
	}
	if err := stack.SetDeviceFault("portal-host", devicestate.FaultCaptivePortal, 1); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/status", "/generate_204", "/unknown?probe=1"} {
		got := readPortalWire(t, "10.254.200.11"+path)
		if got.status != http.StatusFound || got.header.Get("Location") != "/niac-portal" ||
			got.body != "" {
			t.Fatalf("%s did not redirect to local portal: %+v", path, got)
		}
	}
	landing := readPortalWire(t, "10.254.200.11/niac-portal")
	if landing.status != http.StatusOK || landing.header.Get("Location") != "" ||
		!strings.Contains(landing.body, "Captive portal") {
		t.Fatalf("portal landing failed or loops: %+v", landing)
	}
	if got := readPortalWire(t, "10.254.200.12/status"); !reflect.DeepEqual(got, peer) {
		t.Fatalf("healthy peer changed: %+v, want %+v", got, peer)
	}
	if err := stack.SetDeviceFault("portal-host", devicestate.FaultCaptivePortal, 0); err != nil {
		t.Fatal(err)
	}
	if got := readPortalWire(t, "10.254.200.11/status"); !reflect.DeepEqual(got, baseline) {
		t.Fatalf("clear changed authored response: %+v, want %+v", got, baseline)
	}
	t.Log(
		"actual TCP: local302, landing200, unchanged peer and exact authored response after clear",
	)
}

func startPortalStack(t *testing.T) *protocols.Stack {
	t.Helper()
	engine, err := capture.New(simIface, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)
	cfg := &config.Config{Devices: []config.Device{
		portalWireDevice("portal-host", "10.254.200.11", 11),
		portalWireDevice("peer", "10.254.200.12", 12),
	}}
	stack := protocols.NewStack(engine, cfg, logging.NewDebugConfig(0))
	if err = stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stack.Stop)
	return stack
}

func portalWireDevice(name, address string, suffix byte) config.Device {
	return config.Device{
		Name: name, Type: "server", MACAddress: net.HardwareAddr{2, 0, 0, 0, 0, suffix},
		IPAddresses: []net.IP{net.ParseIP(address)},
		HTTPConfig: &config.HTTPConfig{
			Enabled:    true,
			ServerName: "portal-test",
			Endpoints: []config.HTTPEndpoint{{
				Path: "/status", StatusCode: http.StatusCreated, ContentType: "text/plain", Body: "healthy " + name,
			}},
		},
	}
}

type portalWireResponse struct {
	status int
	header http.Header
	body   string
}

func readPortalWire(t *testing.T, target string) portalWireResponse {
	t.Helper()
	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport, Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Get("http://" + target)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	// Date is the only response field tied to the manager's request time.
	response.Header.Del("Date")
	return portalWireResponse{
		status: response.StatusCode,
		header: response.Header,
		body:   string(body),
	}
}
