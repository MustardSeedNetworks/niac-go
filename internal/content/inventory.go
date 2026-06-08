package content

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// Inventory is the per-kind summary `niac content list` prints. One
// entry per library kind, even when zero files are present, so the
// output table always has the same shape.
type Inventory struct {
	Root  string
	Kinds []KindStats
	Total KindStats
}

// KindStats is the count + size of files under a single library kind.
type KindStats struct {
	Kind  library.Kind
	Files int
	Bytes int64
}

// Scan walks the library at root and returns per-kind file counts and
// byte totals. A missing kind subdir is treated as empty; any other
// error surfaces so the caller knows the inventory is incomplete
// instead of silently under-reporting.
func Scan(root string) (Inventory, error) {
	abs, absErr := filepath.Abs(root)
	if absErr != nil {
		return Inventory{}, fmt.Errorf("resolve library root: %w", absErr)
	}

	inv := Inventory{Root: abs}
	for _, kind := range library.AllKinds() {
		stats := KindStats{Kind: kind}
		dir := filepath.Join(abs, string(kind))

		walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, perEntryErr error) error {
			if perEntryErr != nil {
				if os.IsNotExist(perEntryErr) && path == dir {
					// Library subdir hasn't been created yet — treat as empty.
					return filepath.SkipDir
				}
				return fmt.Errorf("walk entry %s: %w", path, perEntryErr)
			}
			if d.IsDir() {
				return nil
			}
			info, infoErr := d.Info()
			if infoErr != nil {
				return fmt.Errorf("stat entry %s: %w", path, infoErr)
			}
			stats.Files++
			stats.Bytes += info.Size()
			return nil
		})
		if walkErr != nil {
			return inv, fmt.Errorf("walk %s: %w", dir, walkErr)
		}

		inv.Kinds = append(inv.Kinds, stats)
		inv.Total.Files += stats.Files
		inv.Total.Bytes += stats.Bytes
	}
	return inv, nil
}

// HumanBytes renders a byte count in a compact human-friendly form
// (1.2 GB, 340 MB, 18 KB). Used by the CLI list output. Kept here so
// both the CLI and any future API handler share the same formatting.
func HumanBytes(n int64) string {
	suffixes := []struct {
		threshold int64
		unit      string
	}{
		{1 << 40, "TB"},
		{1 << 30, "GB"},
		{1 << 20, "MB"},
		{1 << 10, "KB"},
	}
	for _, s := range suffixes {
		if n >= s.threshold {
			return fmt.Sprintf("%.1f %s", float64(n)/float64(s.threshold), s.unit)
		}
	}
	return fmt.Sprintf("%d B", n)
}
