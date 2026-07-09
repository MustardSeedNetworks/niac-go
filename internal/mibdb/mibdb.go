// Package mibdb provides SQLite-based MIB database management for SNMP OID resolution
// and VariMib handler configuration. The database is embedded in the binary and can
// be extended with walk files or custom OID entries.
package mibdb

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

// Time conversion constants.
const (
	// msPerCentisecond is the number of milliseconds per centisecond.
	msPerCentisecond = 10
)

// Sentinel errors for mibdb package.
var (
	// ErrNotFound is returned when a requested entry does not exist in the database.
	ErrNotFound = errors.New("entry not found")
	// ErrUnknownOIDName is returned when an OID name cannot be resolved.
	ErrUnknownOIDName = errors.New("unknown OID name")
	// ErrInvalidVarimibIntegralFormat is returned when the varimib integral config is malformed.
	ErrInvalidVarimibIntegralFormat = errors.New("invalid varimib integral format")
	// ErrInvalidCentiseconds is returned when centiseconds cannot be parsed.
	ErrInvalidCentiseconds = errors.New("invalid centiseconds")
	// ErrInvalidDelta is returned when delta cannot be parsed.
	ErrInvalidDelta = errors.New("invalid delta")
	// ErrInvalidVarimibStringFormat is returned when the varimib string config is malformed.
	ErrInvalidVarimibStringFormat = errors.New("invalid varimib string format")
)

//go:embed schema.sql
var schemaSQL embed.FS

// DB represents the MIB database.
type DB struct {
	db *sql.DB
	mu sync.RWMutex
}

// OIDEntry represents a named OID mapping.
type OIDEntry struct {
	Name     string // Human-readable name (e.g., "sysDescr")
	OID      string // Numeric OID (e.g., "1.3.6.1.2.1.1.1")
	FullPath string // Full human-readable path (optional)
	MIBName  string // Source MIB name (e.g., "SNMPv2-MIB")
}

// VariMibHandler represents a time-varying MIB handler configuration.
type VariMibHandler struct {
	OIDPattern  string // OID pattern or exact match
	HandlerType string // "integral", "string", "fixed", "sysuptime"
	Config      string // Handler-specific configuration
}

// VariMibIntegralConfig represents configuration for integral (counter) handlers
// Format in config files: varimib((2000 10) (4000 -10))
// Meaning: increment by 10 every 2000 centiseconds, then decrement by 10 every 4000 centiseconds.
type VariMibIntegralConfig struct {
	Intervals []struct {
		CentiSeconds int64 // Interval in centiseconds (1/100 sec)
		Delta        int64 // Value to add each interval
	}
}

// VariMibStringConfig represents configuration for string-cycling handlers
// Format: varimib((42000 10.250.0.3) (42000 10.250.0.2))
// Meaning: cycle through values every 42000 centiseconds.
type VariMibStringConfig struct {
	Intervals []struct {
		CentiSeconds int64  // Interval in centiseconds
		Value        string // Value to use during this interval
	}
}

// New creates a new MIB database, either in-memory or file-based.
func New(dbPath string) (*DB, error) {
	var dsn string
	if dbPath == "" || dbPath == ":memory:" {
		dsn = ":memory:"
	} else {
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}

		dsn = dbPath
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	initErr := initSchema(db)
	if initErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to initialize schema: %w", initErr)
	}

	return &DB{db: db}, nil
}

// NewInMemory creates a new in-memory MIB database pre-populated with standard OIDs.
func NewInMemory() (*DB, error) {
	db, err := New(":memory:")
	if err != nil {
		return nil, err
	}

	// Load built-in OID definitions
	loadErr := db.LoadBuiltinOIDs()
	if loadErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to load builtin OIDs: %w", loadErr)
	}

	return db, nil
}

func initSchema(db *sql.DB) error {
	schema, err := schemaSQL.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = db.ExecContext(context.Background(), string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// AddOID adds a single OID entry to the database.
func (d *DB) AddOID(entry OIDEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO oid_names (name, oid, full_path, mib_name)
		VALUES (?, ?, ?, ?)
	`, entry.Name, entry.OID, entry.FullPath, entry.MIBName)
	if err != nil {
		return fmt.Errorf("failed to add OID: %w", err)
	}

	return nil
}

// AddOIDs adds multiple OID entries in a single transaction.
func (d *DB) AddOIDs(entries []OIDEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ctx := context.Background()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }() // rollback is safe to call after commit, returns ErrTxDone

	stmt, prepErr := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO oid_names (name, oid, full_path, mib_name)
		VALUES (?, ?, ?, ?)
	`)
	if prepErr != nil {
		return fmt.Errorf("failed to prepare statement: %w", prepErr)
	}

	defer func() { _ = stmt.Close() }()

	for _, entry := range entries {
		_, execErr := stmt.ExecContext(ctx, entry.Name, entry.OID, entry.FullPath, entry.MIBName)
		if execErr != nil {
			return fmt.Errorf("failed to execute statement: %w", execErr)
		}
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}
	return nil
}

// ResolveOIDName converts a name-based OID to numeric form
// e.g., "sysDescr.0" -> "1.3.6.1.2.1.1.1.0".
func (d *DB) ResolveOIDName(oid string) (string, error) {
	// If already numeric, return as-is
	if strings.HasPrefix(oid, ".") || (len(oid) > 0 && oid[0] >= '0' && oid[0] <= '9') {
		return strings.TrimPrefix(oid, "."), nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	// Extract the name part (before any dots or instance suffix)
	name := oid
	suffix := ""

	if idx := strings.Index(oid, "."); idx > 0 {
		name = oid[:idx]
		suffix = oid[idx:]
	}

	var numericOID string

	err := d.db.QueryRowContext(context.Background(), `SELECT oid FROM oid_names WHERE name = ?`, name).
		Scan(&numericOID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%w: %s", ErrUnknownOIDName, name)
		}

		return "", fmt.Errorf("failed to query OID: %w", err)
	}

	return numericOID + suffix, nil
}

// GetOIDByName looks up an OID entry by name.
func (d *DB) GetOIDByName(name string) (*OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var entry OIDEntry

	err := d.db.QueryRowContext(context.Background(), `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM oid_names WHERE name = ?
	`, name).Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to query OID by name: %w", err)
	}

	return &entry, nil
}

// GetOIDsByPrefix returns all OIDs that start with the given prefix.
func (d *DB) GetOIDsByPrefix(prefix string) ([]OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, queryErr := d.db.QueryContext(context.Background(), `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM oid_names WHERE oid LIKE ? || '%'
		ORDER BY oid
	`, prefix)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to query OIDs by prefix: %w", queryErr)
	}

	defer func() { _ = rows.Close() }()

	var entries []OIDEntry

	for rows.Next() {
		var entry OIDEntry

		scanErr := rows.Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan row: %w", scanErr)
		}

		entries = append(entries, entry)
	}

	rowsErr := rows.Err()
	if rowsErr != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", rowsErr)
	}
	return entries, nil
}

// AddVariMibHandler adds a VariMib handler configuration.
func (d *DB) AddVariMibHandler(handler VariMibHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO varimib_handlers (oid_pattern, handler_type, config)
		VALUES (?, ?, ?)
	`, handler.OIDPattern, handler.HandlerType, handler.Config)
	if err != nil {
		return fmt.Errorf("failed to add VariMib handler: %w", err)
	}

	return nil
}

// GetVariMibHandler retrieves the handler for a specific OID.
func (d *DB) GetVariMibHandler(oid string) (*VariMibHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var handler VariMibHandler

	err := d.db.QueryRowContext(context.Background(), `
		SELECT oid_pattern, handler_type, config
		FROM varimib_handlers
		WHERE oid_pattern = ? OR ? LIKE oid_pattern
		LIMIT 1
	`, oid, oid).Scan(&handler.OIDPattern, &handler.HandlerType, &handler.Config)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to query VariMib handler: %w", err)
	}

	return &handler, nil
}

// ParseVariMibIntegral parses the varimib integral format: varimib((2000 10) (4000 -10)).
func ParseVariMibIntegral(config string) (*VariMibIntegralConfig, error) {
	// Remove "varimib" prefix and brackets
	config = strings.TrimSpace(config)
	config = strings.TrimPrefix(config, "varimib")
	config = strings.TrimPrefix(config, "[")
	config = strings.TrimSuffix(config, "]")

	// Parse interval pairs
	re := regexp.MustCompile(`\((\d+)\s+(-?\d+)\)`)
	matches := re.FindAllStringSubmatch(config, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidVarimibIntegralFormat, config)
	}

	result := &VariMibIntegralConfig{}

	for _, match := range matches {
		centisecs, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCentiseconds, match[1])
		}

		delta, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidDelta, match[2])
		}

		result.Intervals = append(result.Intervals, struct {
			CentiSeconds int64
			Delta        int64
		}{centisecs, delta})
	}

	return result, nil
}

// ParseVariMibString parses the varimib string format: varimib((42000 10.250.0.3) (42000 10.250.0.2)).
func ParseVariMibString(config string) (*VariMibStringConfig, error) {
	config = strings.TrimSpace(config)
	config = strings.TrimPrefix(config, "varimib")
	config = strings.TrimPrefix(config, "[")
	config = strings.TrimSuffix(config, "]")

	// Parse interval pairs with string values
	re := regexp.MustCompile(`\((\d+)\s+([^)]+)\)`)
	matches := re.FindAllStringSubmatch(config, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidVarimibStringFormat, config)
	}

	result := &VariMibStringConfig{}

	for _, match := range matches {
		centisecs, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCentiseconds, match[1])
		}

		result.Intervals = append(result.Intervals, struct {
			CentiSeconds int64
			Value        string
		}{centisecs, strings.TrimSpace(match[2])})
	}

	return result, nil
}

// VariMibIntegralHandler implements a counter that changes over time.
type VariMibIntegralHandler struct {
	config    *VariMibIntegralConfig
	startTime time.Time
	baseValue int64
}

// NewVariMibIntegralHandler creates a new integral handler.
func NewVariMibIntegralHandler(config *VariMibIntegralConfig, baseValue int64) *VariMibIntegralHandler {
	return &VariMibIntegralHandler{
		config:    config,
		startTime: time.Now(),
		baseValue: baseValue,
	}
}

// Value returns the current value based on elapsed time.
func (h *VariMibIntegralHandler) Value() int64 {
	elapsed := time.Since(h.startTime)
	centisecs := elapsed.Milliseconds() / msPerCentisecond // Convert to centiseconds

	value := h.baseValue

	// For each interval configuration, calculate accumulated changes
	for _, interval := range h.config.Intervals {
		if interval.CentiSeconds <= 0 {
			continue
		}
		// Number of complete intervals that have passed
		count := centisecs / interval.CentiSeconds
		value += count * interval.Delta
	}

	// Ensure non-negative for counters
	value = max(value, 0)

	return value
}

// VariMibStringHandler implements a string that cycles through values.
type VariMibStringHandler struct {
	config    *VariMibStringConfig
	startTime time.Time
}

// NewVariMibStringHandler creates a new string handler.
func NewVariMibStringHandler(config *VariMibStringConfig) *VariMibStringHandler {
	return &VariMibStringHandler{
		config:    config,
		startTime: time.Now(),
	}
}

// Value returns the current string value based on elapsed time.
func (h *VariMibStringHandler) Value() string {
	if len(h.config.Intervals) == 0 {
		return ""
	}

	elapsed := time.Since(h.startTime)
	centisecs := elapsed.Milliseconds() / msPerCentisecond

	// Calculate total cycle time
	var totalCycle int64
	for _, interval := range h.config.Intervals {
		totalCycle += interval.CentiSeconds
	}

	if totalCycle == 0 {
		return h.config.Intervals[0].Value
	}

	// Find position in cycle
	pos := centisecs % totalCycle

	var accumulated int64
	for _, interval := range h.config.Intervals {
		accumulated += interval.CentiSeconds
		if pos < accumulated {
			return interval.Value
		}
	}

	return h.config.Intervals[len(h.config.Intervals)-1].Value
}

// Stats returns database statistics.
func (d *DB) Stats() (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]int)

	ctx := context.Background()

	var count int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oid_names`).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count OID names: %w", err)
	}

	stats["oid_names"] = count

	err = d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM varimib_handlers`).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count VariMib handlers: %w", err)
	}

	stats["varimib_handlers"] = count

	return stats, nil
}
