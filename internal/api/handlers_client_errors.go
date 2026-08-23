package api

import (
	"net/http"
)

// maxClientErrorBatch bounds one flush from the browser's error reporter.
// The reporter batches at MAX_BATCH_SIZE = 10; the ceiling here is slack for
// that, not a contract.
const maxClientErrorBatch = 32

// clientErrorReport is one captured front-end error.
type clientErrorReport struct {
	Message   string `json:"message"`
	Stack     string `json:"stack,omitempty"`
	Context   string `json:"context"`
	Timestamp int64  `json:"timestamp"`
	URL       string `json:"url"`
}

type clientErrorBatch struct {
	Errors []clientErrorReport `json:"errors"`
}

// handleClientErrors records front-end errors reported by the browser.
//
// The UI has always POSTed here (ui/src/utils/error-reporter.ts), but the route
// was never implemented, so every flush 404'd. The reporter's `.catch()` could
// not soften that either — a 404 is a resolved fetch, not a rejection — so the
// only visible effect was a second console error, and no client-side error has
// ever reached the server (D7).
//
// Reports are logged, not stored: they are diagnostic breadcrumbs for an
// operator reading the daemon log, and persisting attacker-influenced strings
// would be a liability without any consumer for them.
func (s *Server) handleClientErrors(w http.ResponseWriter, r *http.Request) {
	var batch clientErrorBatch
	if !decodeJSONStrict(w, r, &batch, MaxRequestBodySize) {
		return
	}

	if len(batch.Errors) > maxClientErrorBatch {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"too many errors in one batch", nil)

		return
	}

	for i := range batch.Errors {
		report := &batch.Errors[i]
		s.logger.WarnContext(r.Context(), "[UI] client error",
			"message", report.Message,
			"context", report.Context,
			"url", report.URL,
			"stack", report.Stack,
		)
	}

	w.WriteHeader(http.StatusNoContent)
}
