package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/storage"
)

func addRuns(t *testing.T, store *storage.Storage, count int) {
	t.Helper()
	for i := range count {
		err := store.AddRun(storage.RunRecord{
			StartedAt:  time.Now().Add(time.Duration(i) * time.Second),
			Interface:  "lo0",
			ConfigName: "retention.yaml",
		})
		if err != nil {
			t.Fatalf("AddRun %d: %v", i, err)
		}
	}
}

// The history had no bound: a daemon that starts a session per CI run grew its
// BoltDB file for the life of the install.
func TestPruneKeepsTheNewestRunsOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	store, err := storage.Open(path, storage.RetainAll)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	addRuns(t, store, 1000)

	deleted, err := store.Prune(100)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if deleted != 900 {
		t.Fatalf("pruned %d records, want 900", deleted)
	}

	records, err := store.ListRuns(10_000)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 100 {
		t.Fatalf("records = %d, want 100", len(records))
	}

	// Newest kept, oldest dropped: the records are keyed by an increasing
	// sequence, so the survivors are the tail.
	if records[0].ID != 1000 {
		t.Fatalf("newest surviving ID = %d, want 1000", records[0].ID)
	}
	for _, rec := range records {
		if rec.ID <= 900 {
			t.Fatalf("record %d survived, want only IDs above 900", rec.ID)
		}
	}
}

func TestPruneIsANoOpBelowTheCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	store, err := storage.Open(path, storage.RetainAll)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	addRuns(t, store, 5)

	deleted, err := store.Prune(100)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("pruned %d records from a store below the cap, want 0", deleted)
	}
}

func TestRetainAllKeepsEverything(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	store, err := storage.Open(path, storage.RetainAll)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	addRuns(t, store, 50)
	if deleted, pruneErr := store.Prune(storage.RetainAll); pruneErr != nil || deleted != 0 {
		t.Fatalf("Prune(storage.RetainAll) = %d, %v; want 0, nil", deleted, pruneErr)
	}
}

// Pruning on open is what bounds a database that grew before retention
// existed, and it is what keeps the file from growing across restarts.
func TestOpenPrunesAnOversizedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")

	store, err := storage.Open(path, storage.RetainAll)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	addRuns(t, store, 1000)
	if closeErr := store.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	reopened, err := storage.Open(path, 100)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	records, err := reopened.ListRuns(10_000)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 100 {
		t.Fatalf("records after reopen = %d, want 100", len(records))
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("database missing after prune: %v", statErr)
	}
}
