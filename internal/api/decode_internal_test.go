// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeFixture struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

func newDecodeRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
}

func TestDecodeJSONStrict_ValidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"alpha","port":8080}`)
	var dst decodeFixture

	if !decodeJSONStrict(w, r, &dst, MaxRequestBodySize) {
		t.Fatalf("expected success, got %d: %s", w.Code, w.Body.String())
	}
	if dst.Name != "alpha" || dst.Port != 8080 {
		t.Errorf("unexpected decode: %+v", dst)
	}
}

func TestDecodeJSONStrict_RejectsUnknownField(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"alpha","port":8080,"extra":"oops"}`)
	var dst decodeFixture

	if decodeJSONStrict(w, r, &dst, MaxRequestBodySize) {
		t.Fatal("expected failure for unknown field")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSONStrict_RejectsMalformedJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":}`)
	var dst decodeFixture

	if decodeJSONStrict(w, r, &dst, MaxRequestBodySize) {
		t.Fatal("expected failure for malformed JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSONStrict_RejectsOversizedBody(t *testing.T) {
	big := strings.Repeat("x", 100)
	body := `{"name":"` + big + `","port":1}`
	w := httptest.NewRecorder()
	r := newDecodeRequest(body)
	var dst decodeFixture

	if decodeJSONStrict(w, r, &dst, 50) {
		t.Fatal("expected failure for oversized body")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
}

func TestDecodeJSONStrict_RejectsTrailingData(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"a","port":1}{"name":"b","port":2}`)
	var dst decodeFixture

	if decodeJSONStrict(w, r, &dst, MaxRequestBodySize) {
		t.Fatal("expected failure for trailing data")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "trailing") {
		t.Errorf("response should mention trailing data: %s", w.Body.String())
	}
}

func TestDecodeJSONStrict_RejectsEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(``)
	var dst decodeFixture

	if decodeJSONStrict(w, r, &dst, MaxRequestBodySize) {
		t.Fatal("expected failure for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSONStrict_ResponseIsValidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`not json`)
	var dst decodeFixture

	_ = decodeJSONStrict(w, r, &dst, MaxRequestBodySize)
	var envelope map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error response is not valid JSON: %v\n  body: %s", err, w.Body.String())
	}
	if envelope["error"] != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %v", envelope["error"])
	}
}

func TestDecodeJSONStrictWith_ExtraAttrsCarried(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"alpha","port":8080}`)
	var dst decodeFixture

	if !decodeJSONStrictWith(w, r, &dst, MaxRequestBodySize, "client_ip", "127.0.0.1") {
		t.Fatalf("expected success, got %d: %s", w.Code, w.Body.String())
	}
	if dst.Name != "alpha" {
		t.Errorf("unexpected decode: %+v", dst)
	}
}
