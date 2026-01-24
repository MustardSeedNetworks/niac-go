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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

// Sentinel errors for storage operations.
var (
	ErrStorageDisabled       = errors.New("storage disabled")
	ErrStorageNotInitialised = errors.New("storage not initialised")
)

const (
	runBucket = "runs"
	// File permissions for database file (owner read/write only).
	dbFilePermissions = 0o600
	// Size of uint64 in bytes for key encoding.
	uint64ByteSize = 8
)

// Storage wraps a BoltDB instance for persisting NIAC run history.
type Storage struct {
	db *bbolt.DB
}

// RunRecord captures a single NIAC execution summary.
type RunRecord struct {
	ID              uint64        `json:"id"               yaml:"id"`
	StartedAt       time.Time     `json:"started_at"       yaml:"started_at"`
	Duration        time.Duration `json:"duration"         yaml:"duration"`
	Interface       string        `json:"interface"        yaml:"interface"`
	ConfigName      string        `json:"config_name"      yaml:"config_name"`
	DeviceCount     int           `json:"device_count"     yaml:"device_count"`
	PacketsSent     uint64        `json:"packets_sent"     yaml:"packets_sent"`
	PacketsReceived uint64        `json:"packets_received" yaml:"packets_received"`
	Errors          uint64        `json:"errors"           yaml:"errors"`
}

// Open opens (or creates) the storage database at the requested path.
func Open(path string) (*Storage, error) {
	if strings.EqualFold(path, "disabled") || path == "" {
		return nil, ErrStorageDisabled
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	db, err := bbolt.Open(path, dbFilePermissions, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	initErr := db.Update(func(tx *bbolt.Tx) error {
		_, bucketErr := tx.CreateBucketIfNotExists([]byte(runBucket))
		if bucketErr != nil {
			return fmt.Errorf("failed to create bucket: %w", bucketErr)
		}
		return nil
	})
	if initErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to initialize database: %w", initErr)
	}

	return &Storage{db: db}, nil
}

// Close closes the underlying database.
func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// AddRun stores a run record.
// BoltDB serializes writes internally. The caller (API handler) provides its
// own request-scoped timeout, so we do not wrap in a goroutine which would
// leak if the context expired before the transaction completed.
func (s *Storage) AddRun(record RunRecord) error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(runBucket))
		id, _ := bucket.NextSequence()
		record.ID = id

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to marshal record: %w", err)
		}

		return bucket.Put(itob(id), data)
	})
}

// ListRuns returns the most recent run records up to the requested limit.
// BoltDB serializes reads internally. The caller provides timeout via request context.
func (s *Storage) ListRuns(limit int) ([]RunRecord, error) {
	if s == nil || s.db == nil {
		return nil, ErrStorageNotInitialised
	}

	if limit <= 0 {
		limit = 20
	}

	records := make([]RunRecord, 0, limit)

	err := s.db.View(func(tx *bbolt.Tx) error {
		cursor := tx.Bucket([]byte(runBucket)).Cursor()

		for key, value := cursor.Last(); key != nil && len(records) < limit; key, value = cursor.Prev() {
			var rec RunRecord
			if unmarshalErr := json.Unmarshal(value, &rec); unmarshalErr != nil {
				return fmt.Errorf("failed to unmarshal record: %w", unmarshalErr)
			}

			records = append(records, rec)
		}

		return nil
	})

	return records, err
}

func itob(v uint64) []byte {
	b := make([]byte, uint64ByteSize)

	for i := range uint64ByteSize {
		b[uint64ByteSize-1-i] = byte(v >> (i * uint64ByteSize))
	}

	return b
}
