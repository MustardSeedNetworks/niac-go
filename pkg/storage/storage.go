package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

const runBucket = "runs"

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

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
func (s *Storage) AddRun(record RunRecord) error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(runBucket))
		id, _ := b.NextSequence()
		record.ID = id

		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		return b.Put(itob(id), data)
	})
}

// ListRuns returns the most recent run records up to the requested limit.
func (s *Storage) ListRuns(limit int) ([]RunRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("storage not initialised")
	}
	if limit <= 0 {
		limit = 20
	}

	records := make([]RunRecord, 0, limit)
	err := s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte(runBucket)).Cursor()
		for k, v := c.Last(); k != nil && len(records) < limit; k, v = c.Prev() {
			var rec RunRecord
			if err := json.Unmarshal(v, &rec); err != nil {
				return err
			}
			records = append(records, rec)
		}
		return nil
	})
	return records, err
}

func itob(v uint64) []byte {
	var b [8]byte
	for i := uint(0); i < 8; i++ {
		b[7-i] = byte(v >> (i * 8))
	}
	return b[:]
}
