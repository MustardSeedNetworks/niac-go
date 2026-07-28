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

// ListDrafts returns saved drafts in stable name order.
func (l *Library) ListDrafts() ([]DraftEntry, error) {
	l.draftMu.RLock()
	defer l.draftMu.RUnlock()
	release, lockErr := l.acquireDraftLock(false)
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()

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

	l.draftMu.RLock()
	defer l.draftMu.RUnlock()
	release, lockErr := l.acquireDraftLock(false)
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

	l.draftMu.Lock()
	defer l.draftMu.Unlock()
	release, lockErr := l.acquireDraftLock(true)
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

	l.draftMu.Lock()
	defer l.draftMu.Unlock()
	release, lockErr := l.acquireDraftLock(true)
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

	l.draftMu.Lock()
	defer l.draftMu.Unlock()
	release, lockErr := l.acquireDraftLock(true)
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

func (l *Library) acquireDraftLock(exclusive bool) (func(), error) {
	file, err := l.draftRoot.OpenFile(".lock", os.O_RDWR, draftFileMode)
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
