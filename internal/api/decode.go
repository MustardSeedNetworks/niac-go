// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
)

// decodeJSONStrict reads and validates JSON from r.Body into dst. It applies
// [http.MaxBytesReader] to cap memory, DisallowUnknownFields to reject typoed
// keys, and rejects trailing data after the JSON object.
//
// On any decode failure it writes a structured error response via writeError
// (413 for oversized bodies, 400 for everything else) and returns false; the
// caller should return immediately. On success it returns true and dst is
// populated.
//
// Pass maxSize as a per-call-site limit. Most handlers should pass
// MaxRequestBodySize; specialized endpoints (e.g., bulk-import) may want a
// larger cap. Never pass a magic number.
//
// Mirrors the pattern in seed/internal/api/decode.go and
// stem/internal/api/decode.go (Zod-replacement parity issue #718).
func decodeJSONStrict(w http.ResponseWriter, r *http.Request, dst any, maxSize int64) bool {
	return decodeJSONStrictWith(w, r, dst, maxSize)
}

// decodeJSONStrictWith is the variant of decodeJSONStrict that carries extra
// structured log fields into the WARN line on decode failure. Use for handlers
// where preserving an audit-log breadcrumb matters (e.g., client_ip, device_id,
// session_id).
func decodeJSONStrictWith(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	maxSize int64,
	extraAttrs ...any,
) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	logger := slog.Default()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			attrs := append([]any{"max_size", maxSize}, extraAttrs...)
			logger.WarnContext(r.Context(), "Request body too large", attrs...)
			limit := content.HumanBytes(maxSize)
			writeError(w, r, http.StatusRequestEntityTooLarge,
				"request_too_large",
				fmt.Sprintf("Request body exceeds the %s size limit", limit),
				[]ErrorDetail{{Field: "body", Issue: "max_size_exceeded", Value: limit}})
			return false
		}
		attrs := append([]any{"error", err}, extraAttrs...)
		logger.WarnContext(r.Context(), "Invalid request body", attrs...)
		// Don't leak json parser internals to clients.
		writeError(w, r, http.StatusBadRequest,
			"invalid_request", "Invalid JSON in request body", nil)
		return false
	}

	// Reject "smuggled" data after the top-level JSON object — multiple
	// concatenated payloads, trailing garbage, etc.
	if decoder.More() {
		logger.WarnContext(r.Context(), "Unexpected trailing data after JSON object", extraAttrs...)
		writeError(w, r, http.StatusBadRequest,
			"invalid_request", "Unexpected trailing data after JSON object", nil)
		return false
	}

	return true
}
