// Package storage provides persistent storage for NIAC run history using BoltDB.
//
// The storage layer records execution metadata for each NIAC run including:
//   - Start time and duration
//   - Network interface and configuration used
//   - Device count and packet statistics
//   - Error counts
//
// Features:
//   - BoltDB-based key-value storage
//   - Automatic bucket creation
//   - Sequential run ID generation
//   - Configurable via storage path (can be disabled)
//   - 5-second timeout on all operations to prevent API hangs
//
// Usage:
//
//	storage, err := storage.Open("/path/to/niac.db")
//	if err != nil {
//	    // Handle error or storage disabled
//	}
//	defer storage.Close()
//
//	// Record a run
//	storage.AddRun(storage.RunRecord{
//	    StartedAt: time.Now(),
//	    Interface: "en0",
//	    // ... other fields
//	})
//
//	// Retrieve recent runs
//	runs, err := storage.ListRuns(20)
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

const (
	runBucket = "runs"
	// SECURITY FIX MEDIUM-2: Database operation timeout
	dbOperationTimeout = 5 * time.Second
)

// Storage wraps a BoltDB instance for persisting NIAC run history.
type Storage struct {
	db *bbolt.DB
}

// RunRecord captures a single NIAC execution summary.
type RunRecord struct {
	ID              uint64        `json:"id" yaml:"id"`
	StartedAt       time.Time     `json:"started_at" yaml:"started_at"`
	Duration        time.Duration `json:"duration" yaml:"duration"`
	Interface       string        `json:"interface" yaml:"interface"`
	ConfigName      string        `json:"config_name" yaml:"config_name"`
	DeviceCount     int           `json:"device_count" yaml:"device_count"`
	PacketsSent     uint64        `json:"packets_sent" yaml:"packets_sent"`
	PacketsReceived uint64        `json:"packets_received" yaml:"packets_received"`
	Errors          uint64        `json:"errors" yaml:"errors"`
}

// Open opens (or creates) the storage database at the requested path.
func Open(path string) (*Storage, error) {
	if strings.EqualFold(path, "disabled") || path == "" {
		return nil, errors.New("storage disabled")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}

	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(runBucket))
		return err
	}); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Storage{db: db}, nil
}

// Close closes the underlying database.
func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// AddRun stores a run record.
// SECURITY FIX MEDIUM-2: Add timeout to prevent API hangs on slow storage
func (s *Storage) AddRun(record RunRecord) error {
	if s == nil || s.db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.db.Update(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(runBucket))
			id, _ := bucket.NextSequence()
			record.ID = id

			data, err := json.Marshal(record)
			if err != nil {
				return err
			}
			return bucket.Put(itob(id), data)
		})
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("database operation timeout after %v: %w", dbOperationTimeout, ctx.Err())
	}
}

// ListRuns returns the most recent run records up to the requested limit.
// SECURITY FIX MEDIUM-2: Add timeout to prevent API hangs on slow storage
func (s *Storage) ListRuns(limit int) ([]RunRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("storage not initialised")
	}
	if limit <= 0 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	type result struct {
		records []RunRecord
		err     error
	}
	resultChan := make(chan result, 1)

	go func() {
		records := make([]RunRecord, 0, limit)
		err := s.db.View(func(tx *bbolt.Tx) error {
			cursor := tx.Bucket([]byte(runBucket)).Cursor()
			for key, value := cursor.Last(); key != nil && len(records) < limit; key, value = cursor.Prev() {
				var rec RunRecord
				if err := json.Unmarshal(value, &rec); err != nil {
					return err
				}
				records = append(records, rec)
			}
			return nil
		})
		resultChan <- result{records: records, err: err}
	}()

	select {
	case res := <-resultChan:
		return res.records, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("database operation timeout after %v: %w", dbOperationTimeout, ctx.Err())
	}
}

func itob(v uint64) []byte {
	var b [8]byte
	for i := uint(0); i < 8; i++ {
		b[7-i] = byte(v >> (i * 8))
	}
	return b[:]
}
