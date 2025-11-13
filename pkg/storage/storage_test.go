package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorageAddAndListRuns(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "runs.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})

	rec1 := RunRecord{
		StartedAt:       time.Now().Add(-1 * time.Hour),
		Duration:        time.Minute,
		Interface:       "en0",
		ConfigName:      "test.yaml",
		DeviceCount:     3,
		PacketsSent:     100,
		PacketsReceived: 90,
		Errors:          1,
	}
	rec2 := RunRecord{
		StartedAt:       time.Now(),
		Duration:        2 * time.Minute,
		Interface:       "en1",
		ConfigName:      "test2.yaml",
		DeviceCount:     5,
		PacketsSent:     200,
		PacketsReceived: 180,
		Errors:          0,
	}

	if err := store.AddRun(rec1); err != nil {
		t.Fatalf("AddRun(rec1) error = %v", err)
	}
	if err := store.AddRun(rec2); err != nil {
		t.Fatalf("AddRun(rec2) error = %v", err)
	}

	records, err := store.ListRuns(0) // exercise default limit handling
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListRuns() len = %d, want 2", len(records))
	}
	if records[0].Interface != rec2.Interface || records[0].ID != 2 {
		t.Fatalf("ListRuns() first record = %+v, want latest run with ID 2", records[0])
	}
	if records[1].Interface != rec1.Interface || records[1].ID != 1 {
		t.Fatalf("ListRuns() second record = %+v, want oldest run with ID 1", records[1])
	}
}

func TestOpenDisabled(t *testing.T) {
	t.Parallel()

	if _, err := Open("disabled"); err == nil {
		t.Fatalf("Open(\"disabled\") expected error, got nil")
	}
}
