package library

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	draftsDirName = "drafts"
	draftDirMode  = 0o700
	draftFileMode = 0o600
)

// ErrRevisionMismatch indicates that a draft changed after the caller read it.
var ErrRevisionMismatch = errors.New("draft revision mismatch")

// DraftEntry is the metadata returned when listing saved drafts.
type DraftEntry struct {
	Name       string    `json:"name"`
	Revision   string    `json:"revision"`
	ModifiedAt time.Time `json:"modifiedAt"`
	SizeBytes  int64     `json:"sizeBytes"`
}

// Draft is a revisioned YAML configuration that is not attached to runtime.
type Draft struct {
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	Format     string    `json:"format"`
	Revision   string    `json:"revision"`
	ModifiedAt time.Time `json:"modifiedAt"`
	SizeBytes  int64     `json:"sizeBytes"`
}

func (l *Library) draftsDir() string {
	return filepath.Join(l.root, draftsDirName)
}

func draftFilename(name string) (string, string, error) {
	trimmed := trimYAMLExt(name)
	if validationErr := validateName(trimmed); validationErr != nil {
		return "", "", validationErr
	}
	return trimmed, trimmed + ".yaml", nil
}

func draftRevision(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

// draftNameLocks hands out one *sync.RWMutex per draft name, so operations
// on different drafts never block each other in-process.
//
// This used to be a single library-wide sync.RWMutex (plus a single
// library-wide flock, see acquireDraftLock): every CreateDraft/ReplaceDraft/
// DeleteDraft call — regardless of which draft it named — serialized behind
// the same lock. Under concurrent load (several E2E specs each authoring
// their own scenario against one daemon) that turned unrelated drafts'
// mutations into a queue, and a request stuck behind enough queued writers
// could exceed a test's timeout despite touching a draft nothing else was
// using (niac-go#1773). Scoping the lock to the draft name — which is
// already the unit of the optimistic-concurrency revision check — removes
// that cross-draft contention without changing same-name semantics.
//
// The map is never pruned: a *sync.RWMutex per draft name that has ever
// existed persists for the life of the process. That is a bounded, small
// cost (tens of bytes per name) against the churn a single daemon sees, and
// far cheaper than adding eviction with its own races.
type draftNameLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.RWMutex
}

func (d *draftNameLocks) forName(name string) *sync.RWMutex {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.locks == nil {
		d.locks = make(map[string]*sync.RWMutex)
	}
	lock, ok := d.locks[name]
	if !ok {
		lock = &sync.RWMutex{}
		d.locks[name] = lock
	}
	return lock
}

// ListDrafts returns saved drafts in stable name order.
//
// Unlike the single-draft operations, listing touches every name at once
// and so cannot be scoped to one name's lock. It takes no lock at all:
// each draft file is read via readDraftUnlocked, which only ever sees a
// fully-written revision (writeDraftAtomic always writes-then-renames), so
// a concurrent writer cannot produce a torn read. The one interleaving this
// allows — a draft named by ReadDir being deleted before it is opened — is
// handled explicitly below rather than by locking the whole directory
// against every writer for the duration of a list.
func (l *Library) ListDrafts() ([]DraftEntry, error) {
	entries, err := fs.ReadDir(l.draftRoot.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("read drafts: %w", err)
	}

	drafts := make([]DraftEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFilename(entry.Name()) {
			continue
		}
		draft, readErr := l.readDraftUnlocked(trimYAMLExt(entry.Name()), entry.Name())
		if errors.Is(readErr, ErrNotFound) {
			// Deleted by a concurrent DeleteDraft between the ReadDir above
			// and this open. Not this list's entry to report.
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		drafts = append(drafts, DraftEntry{
			Name:       draft.Name,
			Revision:   draft.Revision,
			ModifiedAt: draft.ModifiedAt,
			SizeBytes:  draft.SizeBytes,
		})
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].Name < drafts[j].Name })
	return drafts, nil
}

// ReadDraft returns one saved draft and its current content revision.
func (l *Library) ReadDraft(name string) (*Draft, error) {
	trimmed, leaf, err := draftFilename(name)
	if err != nil {
		return nil, err
	}

	lock := l.draftMu.forName(trimmed)
	lock.RLock()
	defer lock.RUnlock()
	release, lockErr := l.acquireDraftLock(trimmed, false)
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()
	return l.readDraftUnlocked(trimmed, leaf)
}

func (l *Library) readDraftUnlocked(trimmed, leaf string) (*Draft, error) {
	file, err := l.draftRoot.Open(leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, trimmed)
		}
		return nil, fmt.Errorf("open draft %s: %w", trimmed, err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat draft %s: %w", trimmed, err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read draft %s: %w", trimmed, err)
	}
	return &Draft{
		Name:       trimmed,
		Content:    string(content),
		Format:     "yaml",
		Revision:   draftRevision(content),
		ModifiedAt: info.ModTime().UTC(),
		SizeBytes:  info.Size(),
	}, nil
}

// CreateDraft saves a new draft without replacing an existing one.
func (l *Library) CreateDraft(name, content string) (*Draft, error) {
	trimmed, leaf, err := draftFilename(name)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, ErrEmptyContent
	}

	lock := l.draftMu.forName(trimmed)
	lock.Lock()
	defer lock.Unlock()
	release, lockErr := l.acquireDraftLock(trimmed, true)
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()

	if _, statErr := l.draftRoot.Stat(leaf); statErr == nil {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, trimmed)
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return nil, fmt.Errorf("stat draft %s: %w", trimmed, statErr)
	}
	if writeErr := l.writeDraftAtomic(leaf, []byte(content)); writeErr != nil {
		return nil, writeErr
	}
	return l.readDraftUnlocked(trimmed, leaf)
}

// ReplaceDraft atomically replaces a draft when expectedRevision is current.
func (l *Library) ReplaceDraft(name, expectedRevision, content string) (*Draft, error) {
	trimmed, leaf, err := draftFilename(name)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, ErrEmptyContent
	}

	lock := l.draftMu.forName(trimmed)
	lock.Lock()
	defer lock.Unlock()
	release, lockErr := l.acquireDraftLock(trimmed, true)
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()

	current, err := l.readDraftUnlocked(trimmed, leaf)
	if err != nil {
		return nil, err
	}
	if current.Revision != expectedRevision {
		return nil, fmt.Errorf("%w: %s", ErrRevisionMismatch, trimmed)
	}
	if writeErr := l.writeDraftAtomic(leaf, []byte(content)); writeErr != nil {
		return nil, writeErr
	}
	return l.readDraftUnlocked(trimmed, leaf)
}

// DeleteDraft removes a draft when expectedRevision is current.
func (l *Library) DeleteDraft(name, expectedRevision string) error {
	trimmed, leaf, err := draftFilename(name)
	if err != nil {
		return err
	}

	lock := l.draftMu.forName(trimmed)
	lock.Lock()
	defer lock.Unlock()
	release, lockErr := l.acquireDraftLock(trimmed, true)
	if lockErr != nil {
		return lockErr
	}
	defer release()

	current, err := l.readDraftUnlocked(trimmed, leaf)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return fmt.Errorf("%w: %s", ErrRevisionMismatch, trimmed)
	}
	if removeErr := l.draftRoot.Remove(leaf); removeErr != nil {
		return fmt.Errorf("delete draft %s: %w", trimmed, removeErr)
	}
	return l.syncDraftDirectory()
}

func (l *Library) writeDraftAtomic(leaf string, content []byte) error {
	tmpLeaf, tmp, err := l.createDraftTemp()
	if err != nil {
		return fmt.Errorf("create draft temp file: %w", err)
	}
	defer func() { _ = l.draftRoot.Remove(tmpLeaf) }()

	if _, writeErr := tmp.Write(content); writeErr != nil {
		_ = tmp.Close()
		return fmt.Errorf("write draft temp file: %w", writeErr)
	}
	if syncErr := tmp.Sync(); syncErr != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync draft temp file: %w", syncErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		return fmt.Errorf("close draft temp file: %w", closeErr)
	}
	if renameErr := l.draftRoot.Rename(tmpLeaf, leaf); renameErr != nil {
		return fmt.Errorf("replace draft %s: %w", leaf, renameErr)
	}
	return l.syncDraftDirectory()
}

func (l *Library) createDraftTemp() (string, *os.File, error) {
	for range 100 {
		var suffix [16]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", nil, fmt.Errorf("generate draft temp name: %w", err)
		}
		leaf := fmt.Sprintf(".draft-%x", suffix)
		file, err := l.draftRoot.OpenFile(leaf, os.O_RDWR|os.O_CREATE|os.O_EXCL, draftFileMode)
		if err == nil {
			return leaf, file, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, errors.New("allocate unique draft temp file")
}

// acquireDraftLock takes the cross-process (flock) lock for one draft name.
// The lock file is named after the draft and created lazily on first use:
// unlike the pre-2026-09 single library-wide ".lock", there is nothing to
// reserve for a name that doesn't exist yet, and nothing to prune when a
// draft is deleted — an empty, unused lock file left behind costs nothing.
// trimmed has already passed validateName (via draftFilename), so it is
// safe to embed directly in the lock file's name.
func (l *Library) acquireDraftLock(trimmed string, exclusive bool) (func(), error) {
	leaf := "." + trimmed + ".lock"
	file, err := l.openOrCreateDraftLockFile(leaf)
	if err != nil {
		return nil, fmt.Errorf("open draft lock: %w", err)
	}
	unlock, err := lockDraftFile(file, exclusive)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock drafts: %w", err)
	}
	return func() {
		unlock()
		_ = file.Close()
	}, nil
}

// openOrCreateDraftLockFile opens a per-draft lock file, creating it on
// first use.
//
// os.Root.OpenFile(O_CREATE) is not safe to call concurrently from two
// separate *os.Root handles on the same not-yet-existing path: on the very
// first creation it can spuriously fail with ErrNotExist instead of either
// creating the file or finding it already created (reproduced directly
// against os.Root outside this package — two Root handles opened on the
// same directory, racing O_CREATE on a name neither has created yet). This
// only affects the first-ever open of a given lock file; once it exists,
// concurrent O_CREATE opens are fine. The old single library-wide ".lock"
// never hit this because bootstrap created it once, synchronously, before
// any operation could race for it — the per-name lock files created here
// have no such upfront moment, so callers retry through the transient
// ErrNotExist instead.
func (l *Library) openOrCreateDraftLockFile(leaf string) (*os.File, error) {
	const attempts = 20
	var lastErr error
	for range attempts {
		file, err := l.draftRoot.OpenFile(leaf, os.O_RDWR|os.O_CREATE, draftFileMode)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}
