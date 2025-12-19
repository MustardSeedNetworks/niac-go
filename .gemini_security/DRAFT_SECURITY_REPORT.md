**Vulnerability:** Path Traversal / Arbitrary File Creation via `--storage` flag
**Severity:** Medium
**Location:**
  - `niac-go/pkg/storage/storage.go` (L26: `func Open(path string)`)
  - `niac-go/pkg/daemon/daemon.go` (L67: `daemon.storage, err = storage.Open(storagePath)`)
  - `niac-go/cmd/niac/cmd_daemon.go` (L59: `daemonCmd.Flags().StringVar(&daemonOpts.storagePath, "storage", "~/.niac/niac.db", "Path to run history database (use 'disabled' to disable)")`)
**Line Content:**
  - `func Open(path string) (*Storage, error)`
  - `daemon.storage, err = storage.Open(storagePath)`
  - `daemonCmd.Flags().StringVar(&daemonOpts.storagePath, "storage", "~/.niac/niac.db", "Path to run history database (use 'disabled' to disable)")`
**Description:** The `storagePath` variable, which is used to determine the location of the BoltDB file, is populated directly from the `--storage` command-line flag. While the `expandPath` function (which calls `filepath.Clean`) provides some sanitization against `../` sequences, it does not prevent a malicious user from specifying an absolute path or a path relative to the current working directory that could lead to arbitrary file creation or overwriting of sensitive files if the `niac` process has sufficient permissions. This could also lead to a denial of service by filling up disk space in an unintended location.
**Recommendation:** Implement stricter validation for the `storagePath` argument.
1.  **Restrict to a specific directory:** Only allow `storagePath` to be within a predefined, secure data directory (e.g., `~/.niac/data/`).
2.  **Validate file extension:** Ensure the path ends with `.db` or a similar expected extension.
3.  **Prevent absolute paths (unless explicitly allowed and validated):** If absolute paths are allowed, ensure they are within a whitelist of safe directories.
---
