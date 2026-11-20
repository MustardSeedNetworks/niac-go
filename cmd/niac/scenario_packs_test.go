package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const testPackID = "hospital"

func TestScenarioListsEveryPack(t *testing.T) {
	listings := map[string]func(){
		"niac scenario list": runScenarioList,
		"niac list scenarios": func() {
			if err := runListScenarios(&listOptions{root: t.TempDir()}); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, run := range listings {
		t.Run(name, func(t *testing.T) {
			output := captureDrainedStdout(t, run)
			if !strings.Contains(output, "Scenario packs") {
				t.Fatalf("no pack heading:\n%s", output)
			}
			_, section, _ := strings.Cut(output, "Scenario packs")
			for _, pack := range scenario.Packs() {
				want := fmt.Sprintf("%d devices", pack.Manifest.DeviceCount)
				line := lineWithPrefix(section, "  "+pack.ID+" ")
				if line == "" || !strings.Contains(line, want) || !strings.Contains(line, pack.Description) {
					t.Errorf("pack %s line = %q, want %q and its description", pack.ID, line, want)
				}
			}
		})
	}
}

// captureDrainedStdout reads the pipe while fn runs. captureStdout reads only
// afterwards, which deadlocks once a pack's multi-megabyte YAML fills the pipe.
func captureDrainedStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writeEnd
	defer func() { os.Stdout = original }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(readEnd)
		done <- buf.String()
	}()
	fn()
	_ = writeEnd.Close()

	return <-done
}

func lineWithPrefix(text, substr string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, substr) {
			return line
		}
	}

	return ""
}

func runScenarioGenerateCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	command := newScenarioGenerateCommand()
	command.SetArgs(args)
	command.SilenceUsage = true
	command.SilenceErrors = true
	var err error
	output := captureDrainedStdout(t, func() { err = command.ExecuteContext(context.Background()) })

	return output, err
}

func TestScenarioGenerateWritesYAML(t *testing.T) {
	t.Setenv("NIAC_API_URL", "")
	target := filepath.Join(t.TempDir(), "hospital.yaml")

	tests := []struct {
		name    string
		args    []string
		wantOut string
	}{
		{name: "stdout", args: []string{testPackID}, wantOut: "devices:"},
		{name: "file", args: []string{testPackID, "-o", target}, wantOut: "Wrote"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runScenarioGenerateCLI(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, tt.wantOut) {
				t.Errorf("output lacks %q (%d bytes)", tt.wantOut, len(output))
			}
		})
	}
	data, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(data), "devices:") {
		t.Errorf("-o file = %d bytes, %v", len(data), err)
	}
}

func TestScenarioGenerateRefusals(t *testing.T) {
	t.Setenv("NIAC_API_URL", "")
	existing := filepath.Join(t.TempDir(), "taken.yaml")
	if err := os.WriteFile(existing, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"refuses to overwrite", []string{testPackID, "-o", existing}, "already exists"},
		{"unknown pack", []string{"nope"}, `unknown scenario pack "nope" (available: hospital`},
		{"output and save", []string{testPackID, "-o", "x.yaml", "--save", "x"}, "mutually exclusive"},
		{"empty save name", []string{testPackID, "--save", ".yaml"}, "needs a network name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runScenarioGenerateCLI(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
	if data, err := os.ReadFile(existing); err != nil || string(data) != "keep" {
		t.Fatalf("existing file was touched: %q, %v", data, err)
	}
}

type fakeLibraryDaemon struct {
	mu      sync.Mutex
	names   []string
	saved   map[string]string
	postErr int
}

func (f *fakeLibraryDaemon) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == "/api/v1/csrf-token":
		_, _ = w.Write([]byte(`{"token":"csrf-1"}`))
	case r.URL.Path == "/api/v1/library/networks" && r.Method == http.MethodGet:
		entries := make([]map[string]string, 0, len(f.names))
		for _, name := range f.names {
			entries = append(entries, map[string]string{"name": name})
		}
		_ = json.NewEncoder(w).Encode(entries)
	case r.URL.Path == "/api/v1/library/networks" && r.Method == http.MethodPost:
		if r.Header.Get("X-Csrf-Token") != "csrf-1" {
			http.Error(w, "no csrf", http.StatusForbidden)
			return
		}
		if f.postErr != 0 {
			http.Error(w, `{"error":"validation_failed"}`, f.postErr)
			return
		}
		var body struct{ Name, Content string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.saved[body.Name] = body.Content
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true}`))
	default:
		http.NotFound(w, r)
	}
}

func TestScenarioGenerateSaveUsesDaemonLibrary(t *testing.T) {
	t.Setenv("NIAC_API_URL", "")
	pack, err := findScenarioPack(testPackID)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		saveAs    string
		existing  []string
		extraArgs []string
		postErr   int
		wantErr   string
	}{
		{name: "new name", saveAs: "clinic"},
		{name: "name with extension", saveAs: "clinic.yaml"},
		{name: "existing refused", saveAs: "clinic", existing: []string{"clinic"}, wantErr: "use --force"},
		{
			name:      "existing replaced with force",
			saveAs:    "clinic",
			existing:  []string{"clinic"},
			extraArgs: []string{"--force"},
		},
		{name: "daemon rejects", saveAs: "clinic", postErr: http.StatusBadRequest, wantErr: "400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemon := &fakeLibraryDaemon{names: tt.existing, saved: map[string]string{}, postErr: tt.postErr}
			server := httptest.NewServer(http.HandlerFunc(daemon.handler))
			defer server.Close()
			args := append([]string{testPackID, "--save", tt.saveAs, "--api", server.URL}, tt.extraArgs...)
			output, runErr := runScenarioGenerateCLI(t, args...)
			if tt.wantErr != "" {
				assertSaveRefused(t, daemon, runErr, tt.wantErr)
				return
			}
			if runErr != nil {
				t.Fatal(runErr)
			}
			assertSaved(t, daemon, output, pack.Request.AttachmentName)
		})
	}
}

// The remedy in the preflight message must be a value the daemon's own flag
// parser accepts and that approves the very binding that was refused.
func TestPolicyDeniedMessageNamesAnApprovingFlag(t *testing.T) {
	tests := []struct {
		name    string
		binding fabric.Binding
	}{
		{"direct", fabric.Binding{Interface: "en16", Mode: fabric.ModeDirect}},
		{"access", fabric.Binding{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200}},
		{"trunk", fabric.Binding{Interface: "eth1", Mode: fabric.ModeTrunk, AccessVLAN: 205}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := fabric.CompilePhysicalBinding(tt.binding)
			var message string
			for _, diagnostic := range report.Diagnostics {
				if diagnostic.Code == fabric.CodeAttachmentPolicyDenied {
					message = diagnostic.Message
				}
			}
			if !strings.Contains(message, `interface "`+tt.binding.Interface+`"`) {
				t.Fatalf("message does not name the interface: %q", message)
			}
			_, after, found := strings.Cut(message, "niac attachment-policy add ")
			if !found {
				t.Fatalf("message names no command: %q", message)
			}
			value, _, _ := strings.Cut(after, "`")
			policy, err := attachpolicy.Parse(value)
			if err != nil {
				t.Fatalf("suggested value %q does not parse: %v", value, err)
			}
			if !policy.Approves(tt.binding) {
				t.Errorf("policy %q does not approve %+v", value, tt.binding)
			}
		})
	}
}

func assertSaved(t *testing.T, daemon *fakeLibraryDaemon, output, attachment string) {
	t.Helper()
	if !strings.Contains(daemon.saved["clinic"], "devices:") {
		t.Fatalf("daemon did not receive the YAML under clinic: %d networks saved", len(daemon.saved))
	}
	want := "--config clinic.yaml --attachment " + attachment + " --attachment-mode direct"
	if !strings.Contains(output, want) || !strings.Contains(output, "niac attachment-policy add <interface>=direct") {
		t.Errorf("follow-up lacks %q:\n%s", want, output)
	}
}

func assertSaveRefused(t *testing.T, daemon *fakeLibraryDaemon, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if daemon.postErr == 0 && len(daemon.saved) != 0 {
		t.Fatalf("saved despite refusal: %v", len(daemon.saved))
	}
}
