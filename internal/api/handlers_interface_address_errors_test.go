package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestInterfaceAddressFaultTargetsDoNotRequireSNMP(t *testing.T) {
	server := createDeviceErrorTestServer(t)
	server.cfg.Config.Devices[1].SNMPConfig.Enabled = new(false)
	server.cfg.Stack = protocols.NewStack(nil, server.cfg.Config, logging.NewDebugConfig(0))
	targets := interfaceFaultTargetsResponse(server.cfg.Stack.InterfaceFaultTargets())
	for _, target := range targets {
		if target.Device != "client1" {
			continue
		}
		if !slices.Equal(target.Interfaces, []string{"Management"}) ||
			!slices.Equal(target.ErrorTypes["Management"], []string{duplicateIPLabel}) {
			t.Fatalf("address-only target advertised incorrect capabilities: %+v", target)
		}
		return
	}
	t.Fatal("address-only target absent")
}

func TestInterfaceAddressFaultAPITypedPayloadAndIndependentClear(t *testing.T) {
	server := createDeviceErrorTestServer(t)
	for _, body := range []string{
		`{"device":"gateway","interface":"Management","errorType":"High Utilization","value":70}`,
		`{"device":"gateway","interface":"Management","errorType":"Duplicate IP","address":"192.168.1.10"}`,
	} {
		rec := httptest.NewRecorder()
		server.handleErrors(rec, httptest.NewRequest(http.MethodPost, "/api/v1/errors", strings.NewReader(body)))
		if rec.Code != http.StatusOK {
			t.Fatalf("POST %s: status=%d %s", body, rec.Code, rec.Body.String())
		}
	}
	read := func() map[string]deviceFaultPayload {
		t.Helper()
		rec := httptest.NewRecorder()
		server.handleErrors(rec, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
		var response struct {
			Active map[string]map[string]map[string]deviceFaultPayload `json:"active_errors"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Active["gateway"]["Management"]
	}
	active := read()
	if payload := active["Duplicate IP"]; payload.Address == nil || *payload.Address != "192.168.1.10" ||
		payload.Value != nil {
		t.Fatalf("incorrect address payload: %+v", active)
	}
	rec := httptest.NewRecorder()
	server.handleErrors(
		rec,
		httptest.NewRequest(
			http.MethodDelete,
			"/api/v1/errors?device=gateway&interface=Management&errorType=Duplicate%20IP",
			nil,
		),
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear status=%d %s", rec.Code, rec.Body.String())
	}
	active = read()
	if payload := active["High Utilization"]; len(active) != 1 || payload.Value == nil || *payload.Value != 70 ||
		payload.Address != nil {
		t.Fatalf("clear changed other fault: %+v", active)
	}
}

func TestInterfaceAddressFaultAPIRejectsInvalidPayload(t *testing.T) {
	for _, fields := range []string{
		`"address":"192.168.1.10"`,
		`"interface":"missing","address":"192.168.1.10"`,
		`"interface":"Management","address":"192.168.1.99"`,
		`"interface":"Management","address":"192.168.1.10","value":0`,
		`"interface":"Management","value":1`,
	} {
		server := createDeviceErrorTestServer(t)
		before := server.cfg.Stack.ExportDeviceStates()
		rec := httptest.NewRecorder()
		server.handleErrors(rec, httptest.NewRequest(http.MethodPost, "/api/v1/errors", strings.NewReader(
			`{"device":"gateway","errorType":"Duplicate IP",`+fields+`}`)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid payload accepted: %s", fields)
		}
		after := server.cfg.Stack.ExportDeviceStates()
		if after["gateway"].Version != before["gateway"].Version {
			t.Fatal("rejected payload changed state")
		}
	}
}
