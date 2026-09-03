package main

// codedError carries a specific process exit code out to the single exit point.
//
// Most commands only need to distinguish success from failure, and a plain
// error is enough. `status` does not: it reports "running" (0), "not running"
// (1) and "could not tell" (2), and scripts branch on the difference. Returning
// a bare error would collapse the last two.
type codedError struct {
	code int
	err  error
}

func (e codedError) Error() string { return e.err.Error() }

func (e codedError) Unwrap() error { return e.err }

// withExitCode tags err with the process exit code it should produce. A nil err
// stays nil, so callers can wrap unconditionally on a path that may have
// succeeded.
func withExitCode(code int, err error) error {
	if err == nil {
		return nil
	}

	return codedError{code: code, err: err}
}
