package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlePcapUploadMethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	// Method gating moved from the handler to the route registry (ADR-0002).
	// Exercise the same methodGate wrapper register() composes for the
	// POST-only /api/v1/pcap/upload route.
	gated := server.methodGate([]string{http.MethodPost}, server.handlePcapUpload)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pcap/upload", nil)
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /pcap/upload: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Errorf("Allow header = %q, want %q", allow, http.MethodPost)
	}
}

func TestHandlePcapUploadMissingData(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload",
		strings.NewReader(`{"filename":"test.pcap","data":""}`))
	server.handlePcapUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /pcap/upload missing data: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapUploadInvalidBase64(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload",
		strings.NewReader(`{"filename":"test.pcap","data":"!!!invalid!!!"}`))
	server.handlePcapUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /pcap/upload invalid base64: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapUploadBadJSON(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload",
		strings.NewReader("not valid json"))
	server.handlePcapUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /pcap/upload bad json: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapAnalysisMethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	// Method gating moved from the handler to the route registry (ADR-0002).
	// Exercise the methodGate wrapper register() composes for the GET-only
	// /api/v1/pcap/ route.
	gated := server.methodGate([]string{http.MethodGet}, server.handlePcapAnalysis)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/test-id", nil)
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /pcap/{id}: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header = %q, want %q", allow, http.MethodGet)
	}
}

func TestHandlePcapAnalysisMissingID(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pcap/", nil)
	server.handlePcapAnalysis(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /pcap/ missing ID: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapAnalysisNotFound(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pcap/nonexistent-id", nil)
	server.handlePcapAnalysis(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /pcap/nonexistent: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPcapUploadResponseJSON(t *testing.T) {
	resp := PcapUploadResponse{
		Success:    true,
		AnalysisID: "abc123",
		Message:    "test message",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded PcapUploadResponse
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.AnalysisID != "abc123" {
		t.Errorf("AnalysisID = %q, want %q", decoded.AnalysisID, "abc123")
	}
}
