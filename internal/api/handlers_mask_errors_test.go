package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaskFaultRequestValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		prefix  *int
		value   *int
		address *string
		valid   bool
	}{
		{name: "zero", prefix: new(0), valid: true},
		{name: "host", prefix: new(32), valid: true},
		{name: "missing"},
		{name: "negative", prefix: new(-1)},
		{name: "too large", prefix: new(33)},
		{name: "numeric", prefix: new(24), value: new(0)},
		{name: "address", prefix: new(24), address: new("192.0.2.1")},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := errorInjectionRequest{
				Device: "host", Interface: "eth0", ErrorType: "Bad Subnet Mask",
				PrefixBits: test.prefix, Value: test.value, Address: test.address,
			}
			if message := request.validationMessage(); (message == "") != test.valid {
				t.Fatalf("validation = %q, want valid %v", message, test.valid)
			}
		})
	}
}

func TestMaskFaultAPIApplyReadClear(t *testing.T) {
	server := createDeviceErrorTestServer(t)
	recorder := httptest.NewRecorder()
	server.handleErrors(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/errors", strings.NewReader(
			`{"device":"client1","interface":"Management","errorType":"Bad Subnet Mask","prefixBits":0}`,
		)),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("apply: %d %s", recorder.Code, recorder.Body.String())
	}
	read := httptest.NewRecorder()
	server.handleErrors(read, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		Active map[string]map[string]map[string]deviceFaultPayload `json:"active_errors"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	payload := response.Active["client1"]["Management"][badMaskLabel]
	if payload.PrefixBits == nil || *payload.PrefixBits != 0 || payload.Value != nil ||
		payload.Address != nil {
		t.Fatalf("wrong active mask: %+v", payload)
	}
	clearResult := httptest.NewRecorder()
	server.handleErrors(clearResult, httptest.NewRequest(http.MethodDelete,
		"/api/v1/errors?device=client1&interface=Management&errorType=Bad%20Subnet%20Mask", nil))
	if clearResult.Code != http.StatusOK ||
		len(server.cfg.Stack.ActiveInterfacePrefixFaults()) != 0 {
		t.Fatalf("clear: %d %s", clearResult.Code, clearResult.Body.String())
	}
}
