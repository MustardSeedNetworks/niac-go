package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerPassesMatchingTopology(t *testing.T) {
	server := linkLiveServer(t, "Switch", "Full", "100 Gb")
	configPath := writeConfig(t)
	var output bytes.Buffer
	executor := newRunner(testEnvironment(server.URL), &output)
	executor.allowInsecure = true

	err := executor.run(
		context.Background(),
		[]string{"-config", configPath, "-analysis", "analysis-7"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"passed": true`) {
		t.Fatalf("output = %s", output.String())
	}
}

func TestRunnerFailsOnMismatch(t *testing.T) {
	server := linkLiveServer(t, "Router", "Unknown", "")
	var output bytes.Buffer
	executor := newRunner(testEnvironment(server.URL), &output)
	executor.allowInsecure = true

	err := executor.run(
		context.Background(),
		[]string{"-config", writeConfig(t), "-analysis", "analysis-7"},
	)
	if err == nil || !strings.Contains(err.Error(), "mismatches") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(output.String(), "type-conflict") {
		t.Fatalf("output = %s", output.String())
	}
}

func linkLiveServer(t *testing.T, deviceType, duplex, speed string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/auth/login" {
			_, _ = w.Write([]byte(`{"accessToken":"token"}`))
			return
		}
		_, _ = w.Write([]byte(topologyJSON(deviceType, duplex, speed)))
	}))
	t.Cleanup(server.Close)
	return server
}

func topologyJSON(deviceType, duplex, speed string) string {
	return `[{"hostId":1,"bestNameFormatted":"COS-CORE-SW01",` +
		`"displayedDeviceType":"` + deviceType + `","longMfrMac":"Cisco:00000c-f00401",` +
		`"defaultAddr":{"ipV4Address":"10.240.200.2"},"connectedHosts":[{` +
		`"connectedHostId":2,"name":"COS-DIST-SW01","mac":"Cisco:00000c-f00501",` +
		`"connectedEdge":{"compiledPort":"HundredGigabitEthernet1/0/1",` +
		`"compiledDuplex":"` + duplex + `","compiledSpeed":"` + speed + `"}}]},` +
		`{"hostId":2,"bestNameFormatted":"COS-DIST-SW01","displayedDeviceType":"Switch",` +
		`"longMfrMac":"Cisco:00000c-f00501","connectedHosts":[]}]`
}

func writeConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.yaml")
	data := []byte(`devices:
  - name: COS-CORE-SW01
    type: layer3-switch
    mac: "00:00:0c:f0:04:01"
    ips: [10.240.200.2]
    interfaces:
      - name: HundredGigabitEthernet1/0/1
        type: ethernet
        mtu: 9000
        speed: 100000
        duplex: full
        admin_status: up
        oper_status: up
        in_utilization: 20
        out_utilization: 30
    trunk_ports:
      - interface: HundredGigabitEthernet1/0/1
        remote_device: COS-DIST-SW01
        remote_interface: HundredGigabitEthernet1/0/1
  - name: COS-DIST-SW01
    type: switch
    mac: "00:00:0c:f0:05:01"
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func testEnvironment(baseURL string) func(string) string {
	return func(key string) string {
		values := map[string]string{
			"LINKLIVE_USERNAME":     "tester@example.com",
			"LINKLIVE_PASSWORD":     "test-password",
			"LINKLIVE_IDENTITY_URL": baseURL,
			"LINKLIVE_API_URL":      baseURL,
		}
		return values[key]
	}
}
