package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddressFaultAPIStoresTypedPayloadAndClearsOnlyNamedFault(t *testing.T) {
	server := createDeviceErrorTestServer(t)
	for _, body := range []string{
		`{"device":"gateway","errorType":"Latency","value":50}`,
		`{"device":"gateway","errorType":"Duplicate DHCP Offer","address":"192.168.1.10"}`,
	} {
		rec := httptest.NewRecorder()
		server.handleErrors(rec, httptest.NewRequest(http.MethodPost, "/api/v1/errors", strings.NewReader(body)))
		if rec.Code != http.StatusOK {
			t.Fatalf("POST status=%d: %s", rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	server.handleErrors(rec, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		Active map[string]map[string]json.RawMessage `json:"active_device_errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Active["gateway"]["Duplicate DHCP Offer"], &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 || string(payload["address"]) != `"192.168.1.10"` {
		t.Fatalf("address payload = %s", response.Active["gateway"]["Duplicate DHCP Offer"])
	}
	rec = httptest.NewRecorder()
	server.handleErrors(rec, httptest.NewRequest(http.MethodDelete,
		"/api/v1/errors?device=gateway&errorType=Duplicate%20DHCP%20Offer", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("clear status=%d: %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	server.handleErrors(rec, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	response.Active = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var remaining deviceFaultPayload
	if err := json.Unmarshal(response.Active["gateway"]["Latency"], &remaining); err != nil {
		t.Fatal(err)
	}
	if len(response.Active["gateway"]) != 1 || remaining.Value == nil || *remaining.Value != 50 ||
		remaining.Address != nil {
		t.Fatalf("clear affected another fault: %s", rec.Body.String())
	}
}

func TestAddressFaultAPIRejectsMixedOrMissingPayload(t *testing.T) {
	for _, fields := range []string{
		`"errorType":"Duplicate DHCP Offer"`,
		`"errorType":"Duplicate DHCP Offer","address":"192.168.1.10","value":0`,
		`"errorType":"Duplicate DHCP Offer","address":"invalid"`,
		`"errorType":"Duplicate DHCP Offer","address":"192.168.1.99"`,
		`"errorType":"Latency"`,
		`"errorType":"Latency","value":5,"address":"192.168.1.10"`,
	} {
		server := createDeviceErrorTestServer(t)
		rec := httptest.NewRecorder()
		server.handleErrors(rec, httptest.NewRequest(http.MethodPost, "/api/v1/errors",
			strings.NewReader(`{"device":"gateway",`+fields+`}`)))
		if rec.Code != http.StatusBadRequest || len(server.cfg.Stack.ActiveDeviceFaults()) != 0 {
			t.Fatalf("invalid payload accepted or changed state: %s: status=%d %s", fields, rec.Code, rec.Body.String())
		}
	}
}
