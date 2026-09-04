package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// daemonStub answers the two endpoints --pcap uses: the session list it
// resolves against, and the export itself.
func daemonStub(t *testing.T, sessions string, capture []byte, seen *string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/simulation":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, sessions)
		case strings.HasSuffix(r.URL.Path, "/capture/export"):
			if seen != nil {
				*seen = r.URL.String()
			}
			w.Header().Set("Content-Type", "application/x-pcapng")
			_, _ = w.Write(capture)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func samplePcapng(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := capturering.WritePcapng(&out, "eth0", []capturering.Frame{
		{Direction: "rx", VLAN: -1, Data: make([]byte, 60)},
	}); err != nil {
		t.Fatalf("WritePcapng: %v", err)
	}

	return out.Bytes()
}

func TestDumpPcapWritesAReadableCaptureForTheOnlyRunningSession(t *testing.T) {
	var requested string
	stub := daemonStub(t, `{"sessions":[{"sessionId":"hospital"}]}`, samplePcapng(t), &requested)
	path := filepath.Join(t.TempDir(), "out.pcapng")

	options := &dumpOptions{api: stub.URL, insecure: true, pcapFile: path, filter: "arp", count: 5}
	if err := runDump(context.Background(), options); err != nil {
		t.Fatalf("runDump: %v", err)
	}

	if !strings.Contains(requested, "hospital") {
		t.Errorf("exported %q, want the hospital session", requested)
	}
	for _, want := range []string{"filter=arp", "last=5"} {
		if !strings.Contains(requested, want) {
			t.Errorf("request %q is missing %q", requested, want)
		}
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	reader, err := pcapgo.NewNgReader(bytes.NewReader(written), pcapgo.DefaultNgReaderOptions)
	if err != nil {
		t.Fatalf("the file is not a readable pcapng: %v", err)
	}
	if _, _, err = reader.ReadPacketData(); err != nil {
		t.Fatalf("ReadPacketData: %v", err)
	}
}

func TestDumpPcapRefusesToGuessBetweenSeveralSessions(t *testing.T) {
	stub := daemonStub(t, `{"sessions":[{"sessionId":"a"},{"sessionId":"b"}]}`, nil, nil)
	path := filepath.Join(t.TempDir(), "out.pcapng")

	err := runDump(context.Background(), &dumpOptions{api: stub.URL, insecure: true, pcapFile: path})
	if err == nil {
		t.Fatal("expected an error naming the ambiguity")
	}
	if !strings.Contains(err.Error(), "--session") {
		t.Errorf("error %q does not point at --session", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("an unresolved session still created the output file")
	}
}

func TestDumpPcapReportsTheWriteAsJSON(t *testing.T) {
	stub := daemonStub(t, `{"sessions":[{"sessionId":"hospital"}]}`, samplePcapng(t), nil)
	path := filepath.Join(t.TempDir(), "out.pcapng")

	stdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = write
	runErr := runDump(context.Background(), &dumpOptions{
		api: stub.URL, insecure: true, pcapFile: path, jsonOutput: true,
	})
	_ = write.Close()
	os.Stdout = stdout
	if runErr != nil {
		t.Fatalf("runDump: %v", runErr)
	}

	var payload struct {
		Success bool   `json:"success"`
		Session string `json:"session"`
		Bytes   int64  `json:"bytes"`
	}
	if err = json.NewDecoder(read).Decode(&payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if !payload.Success || payload.Session != "hospital" || payload.Bytes == 0 {
		t.Errorf("payload = %+v", payload)
	}
}

func TestExportCaptureSendsNoQueryWhenNothingIsNarrowed(t *testing.T) {
	var requested string
	stub := daemonStub(t, "", samplePcapng(t), &requested)
	client, err := cliclient.New(cliclient.Config{BaseURL: stub.URL, Insecure: true})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out bytes.Buffer
	if _, err = client.ExportCapture(
		context.Background(), "hospital", cliclient.CaptureExportOptions{}, &out,
	); err != nil {
		t.Fatalf("ExportCapture: %v", err)
	}
	if strings.Contains(requested, "?") {
		t.Errorf("request %q carried a query string", requested)
	}
}
