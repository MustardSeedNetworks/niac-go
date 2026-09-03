package fabric

import (
	"errors"
	"strings"
)

// ErrUnsafeTopology marks any failure to compile a scenario into a safe
// topology. Callers match on it with errors.Is; UnsafeTopologyError carries
// the findings.
var ErrUnsafeTopology = errors.New("scenario failed to compile into a safe topology")

// UnsafeTopologyError carries the compiler's findings across an error return.
//
// The daemon used to format the diagnostics into the error string, which left
// the API unable to do anything with them but log a fixed code and answer 500
// -- the one response a configuration error must never get, because preflight
// had the same list and could show it. Keeping them structured lets every
// surface report the same codes.
type UnsafeTopologyError struct {
	Diagnostics []Diagnostic
}

// NewUnsafeTopologyError wraps compiler diagnostics as an error.
func NewUnsafeTopologyError(diagnostics []Diagnostic) error {
	return &UnsafeTopologyError{Diagnostics: diagnostics}
}

func (e *UnsafeTopologyError) Error() string {
	issues := make([]string, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		issues = append(issues, string(diagnostic.Code)+": "+diagnostic.Field+": "+diagnostic.Message)
	}
	return ErrUnsafeTopology.Error() + ": " + strings.Join(issues, "; ")
}

// Is reports UnsafeTopologyError as ErrUnsafeTopology so callers that only
// need the classification do not have to unwrap the findings.
func (e *UnsafeTopologyError) Is(target error) bool { return target == ErrUnsafeTopology }
