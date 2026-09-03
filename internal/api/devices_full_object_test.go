package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The UI posts a whole Device object: ui/src/api/client.ts's createDevice takes
// `device: Device` and sends it unmodified. DeviceCreateRequest covers a
// fraction of that type, and the handler decodes strictly, so every field the
// editor can set but the DTO omits turns a save into a 400 the operator cannot
// act on.
//
// One field per subtest rather than one big body, so a failure names the field
// that is missing instead of just "something in here is unknown".
func TestDeviceCreateAcceptsEveryEditableField(t *testing.T) {
	fields := map[string]string{
		"ips":           `"ips": ["10.0.0.9"]`,
		"vlan":          `"vlan": 10`,
		"babble":        `"babble": true`,
		"mapToIp":       `"mapToIp": "10.0.0.99"`,
		"lldp":          `"lldp": {"enabled": true}`,
		"cdp":           `"cdp": {"enabled": true}`,
		"edp":           `"edp": {"enabled": true}`,
		"fdp":           `"fdp": {"enabled": true}`,
		"stp":           `"stp": {"enabled": true}`,
		"dhcp":          `"dhcp": {"enabled": true}`,
		"dns":           `"dns": {"enabled": true}`,
		"http":          `"http": {"enabled": true}`,
		"ftp":           `"ftp": {"enabled": true}`,
		"netbios":       `"netbios": {"enabled": true}`,
		"icmp":          `"icmp": {"enabled": true}`,
		"icmpv6":        `"icmpv6": {"enabled": true}`,
		"dhcpv6":        `"dhcpv6": {"enabled": true}`,
		"ttl":           `"ttl": {"enabled": true}`,
		"osFingerprint": `"osFingerprint": {"enabled": true}`,
		"iperf3":        `"iperf3": {"enabled": true}`,
	}

	for name, fragment := range fields {
		t.Run(name, func(t *testing.T) {
			server := newDeviceTestServer(t)
			body := `{"hostname":"new1","type":"router","mac":"00:11:22:33:44:77",` + fragment + `}`

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.handleDevicesV2(rec, req)

			// A field the DTO does not declare is rejected by the strict decoder
			// as "Invalid JSON in request body" — the message names no field, so
			// match on the status and report the body for diagnosis.
			if rec.Code == http.StatusBadRequest {
				t.Fatalf("%s is editable in the UI but rejected by the create DTO: %d %s",
					name, rec.Code, strings.TrimSpace(rec.Body.String()))
			}
		})
	}
}
