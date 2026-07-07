package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pcapGlobalHeader24 builds a minimal-but-valid 24-byte little-endian pcap
// global header (magic + version 2.4 + Ethernet link type), satisfying
// capture.ValidateStructure without needing well-formed packet records after
// it — ValidateStructure only inspects the fixed-size global header.
func pcapGlobalHeader24() []byte {
	const (
		magicLittleEndian = 0xa1b2c3d4
		versionMajor      = 2
		versionMinor      = 4
		linkTypeEthernet  = 1 // dltEN10MB
	)
	h := make([]byte, 24)
	// Magic number, little-endian byte order.
	h[0], h[1], h[2], h[3] = 0xd4, 0xc3, 0xb2, 0xa1
	// Version major/minor (2.4), little-endian.
	h[4], h[5] = versionMajor, 0
	h[6], h[7] = versionMinor, 0
	// thiszone, sigfigs, snaplen: left zero, unchecked by ValidateStructure.
	// Link-layer type (Ethernet), little-endian, at offset 20.
	h[20], h[21], h[22], h[23] = linkTypeEthernet, 0, 0, 0

	return h
}

// buildPcapUploadJSONBody returns the raw HTTP request body for
// /api/v1/pcap/upload: a JSON envelope wrapping a base64-encoded pcap of
// exactly rawSize bytes (global header + zero-filled padding).
func buildPcapUploadJSONBody(rawSize int) []byte {
	raw := make([]byte, rawSize)
	copy(raw, pcapGlobalHeader24())

	encoded := base64.StdEncoding.EncodeToString(raw)

	var body bytes.Buffer
	body.Grow(len(encoded) + 64)
	body.WriteString(`{"filename":"boundary-test.pcap","data":"`)
	body.WriteString(encoded)
	body.WriteString(`"}`)

	return body.Bytes()
}

// TestPcapUploadBodyCapBoundary is the regression test for bug #1e: the
// pcap upload UI advertises a 100MB (raw) max, but the JSON request body —
// which wraps the base64-encoded pcap — is ~4/3 larger than the raw pcap
// plus a small envelope. The body cap (MaxPCAPUploadBodySize) must be sized
// for that expansion, not reuse the raw cap (MaxPCAPUploadSize) directly.
func TestPcapUploadBodyCapBoundary(t *testing.T) {
	tests := []struct {
		name       string
		rawSize    int
		wantOK     bool
		wantStatus int
	}{
		{
			// A genuine 100MiB raw pcap (the advertised max) must be
			// accepted: its ~137MiB base64 JSON body must fit under
			// MaxPCAPUploadBodySize (140MiB).
			name:    "100MiB raw pcap (advertised max) is accepted",
			rawSize: MaxPCAPUploadSize,
			wantOK:  true,
		},
		{
			// Meaningfully over the raw 100MB cap (120MiB raw -> ~160MiB
			// base64 JSON body) must be rejected via the structured JSON
			// error path, not a bare net/http 413.
			name:       "120MiB raw pcap (over advertised max) is rejected",
			rawSize:    MaxPCAPUploadSize + 20<<20,
			wantOK:     false,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildPcapUploadJSONBody(tt.rawSize)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			pcapData, _, ok := decodePcapUpload(rec, req)
			if ok != tt.wantOK {
				t.Fatalf("decodePcapUpload ok = %v, want %v (status=%d body=%s)",
					ok, tt.wantOK, rec.Code, rec.Body.String())
			}

			if tt.wantOK {
				if len(pcapData) != tt.rawSize {
					t.Errorf("decoded pcap size = %d, want %d", len(pcapData), tt.rawSize)
				}
				return
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			// Assert the rejection is a structured JSON error (writeError),
			// not net/http's bare "request body too large" text -- a
			// regression here would silently strip the JSON error envelope.
			var errResp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("error response is not valid JSON: %v (body=%q)", err, rec.Body.String())
			}
			if errResp.Error == "" {
				t.Errorf("error response missing machine-readable code, got: %+v", errResp)
			}
		})
	}
}

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
