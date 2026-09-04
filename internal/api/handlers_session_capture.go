package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
)

// maxExportLast bounds the ?last= parameter. Anything larger is the ring's
// whole contents anyway, and refusing an absurd value keeps a typo from
// reading as a request the daemon then has to justify.
const maxExportLast = 1_000_000

// handleSessionCaptureExport serves GET /api/v1/sessions/{id}/capture/export.
//
// The Packet Inspector shows frames it can never save, and `niac dump` reads
// the live stream, so anything that happened before the operator looked is
// gone. This writes what the session's ring holds as pcapng — full frames,
// per-frame timestamps, and the fabric decision as a packet comment, which is
// the part no sniffer on the wire could reconstruct.
//
// Query parameters:
//
//	filter — a libpcap BPF expression, applied to the retained frames
//	last   — keep only the newest N frames
func (s *Server) handleSessionCaptureExport(
	w http.ResponseWriter, r *http.Request, session sessionRuntime,
) {
	if !requireGet(w, r) {
		return
	}

	last, ok := exportLast(w, r)
	if !ok {
		return
	}

	// Compile before writing a byte: once the pcapng header is on the wire the
	// status code is spent, and a bad filter would surface as a truncated
	// download rather than a 400.
	match, err := capture.NewEthernetMatcher(r.URL.Query().Get("filter"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_filter", err.Error(),
			[]ErrorDetail{{Field: "filter", Issue: err.Error()}})
		return
	}

	frames := filterFrames(session.captureFrames(last), match)

	w.Header().Set("Content-Type", "application/x-pcapng")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", "niac-"+session.id+".pcapng"))
	w.Header().Set("X-Niac-Frame-Count", strconv.Itoa(len(frames)))

	if err = capturering.WritePcapng(w, session.iface(), frames); err != nil {
		// The header is already sent, so the client sees a short file rather
		// than an error document. Log it so the daemon's operator can.
		s.logger.ErrorContext(r.Context(), "[API] pcapng export failed",
			"error", err, "session", session.id)
	}
}

// exportLast reads and validates ?last=. Absent means "everything".
func exportLast(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("last")
	if raw == "" {
		return 0, true
	}
	last, err := strconv.Atoi(raw)
	if err != nil || last < 0 || last > maxExportLast {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed",
			[]ErrorDetail{{
				Field: "last",
				Issue: fmt.Sprintf("last must be a whole number between 0 and %d", maxExportLast),
			}})

		return 0, false
	}

	return last, true
}

// filterFrames keeps the frames the BPF program matches. It compacts in place:
// frames is the ring's snapshot copy, so reordering it touches nothing the
// ring still owns.
func filterFrames(frames []capturering.Frame, match capture.Matcher) []capturering.Frame {
	kept := frames[:0]
	for _, frame := range frames {
		if match(frame.Data) {
			kept = append(kept, frame)
		}
	}

	return kept
}
