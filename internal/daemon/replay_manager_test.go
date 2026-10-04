package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

type discardSender struct{}

func (discardSender) SendPacket([]byte) error { return nil }

func writeEmptyPCAP(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create pcap: %v", err)
	}
	defer func() { _ = f.Close() }()
	w := pcapgo.NewWriter(f)
	if err = w.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
		t.Fatalf("write pcap header: %v", err)
	}
}

// TestReplayManagerMapsEveryField pins the daemon's mapping between the API
// DTOs and the replay package's types (#2434): every request field reaches
// the controller and every state field comes back.
func TestReplayManagerMapsEveryField(t *testing.T) {
	dir := t.TempDir()
	pcapPath := filepath.Join(dir, "upload.pcap")
	writeEmptyPCAP(t, pcapPath)

	manager := newReplayController(discardSender{}, 0)
	req := api.ReplayRequest{
		File:      pcapPath,
		LoopMs:    250,
		Scale:     1.5,
		RateMode:  "pps",
		Pps:       1000,
		MbpsCap:   50,
		LoopCount: 3,
		BPFFilter: "ip",
		RootDir:   dir,
		Uploaded:  true,
	}
	started, err := manager.Start(req)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	want := api.ReplayState{
		Running:   true,
		File:      req.File,
		LoopMs:    req.LoopMs,
		Scale:     req.Scale,
		RateMode:  req.RateMode,
		Pps:       req.Pps,
		MbpsCap:   req.MbpsCap,
		LoopCount: req.LoopCount,
		BPFFilter: req.BPFFilter,
		StartedAt: started.StartedAt,
	}
	if started.StartedAt.IsZero() || time.Since(started.StartedAt) > time.Minute {
		t.Fatalf("StartedAt = %v, want now", started.StartedAt)
	}
	if started != want {
		t.Fatalf("Start state = %+v, want %+v", started, want)
	}
	if status := manager.Status(); status.File != want.File || !status.Running {
		t.Fatalf("Status = %+v, want running on %s", status, want.File)
	}

	stopped, err := manager.Stop()
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stopped.Running || stopped.File != want.File {
		t.Fatalf("Stop state = %+v, want stopped on %s", stopped, want.File)
	}
	if _, statErr := os.Stat(pcapPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("uploaded pcap survived Stop (Uploaded not mapped): stat err = %v", statErr)
	}
}

// TestReplayManagerMapsRootDir proves RootDir reaches playback: a file that
// escapes the allow-listed root through a symlink opens when playback anchors
// at the file's parent, and is refused when anchored at RootDir.
func TestReplayManagerMapsRootDir(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeEmptyPCAP(t, filepath.Join(outside, "escape.pcap"))
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}
	file := filepath.Join(root, "link", "escape.pcap")

	manager := newReplayController(discardSender{}, 0)
	if _, err := manager.Start(api.ReplayRequest{File: file}); err != nil {
		t.Fatalf("Start without RootDir: %v", err)
	}
	if _, err := manager.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if _, err := manager.Start(api.ReplayRequest{File: file, RootDir: root}); err == nil {
		t.Fatal("Start anchored at RootDir opened a file outside it")
	}
}
