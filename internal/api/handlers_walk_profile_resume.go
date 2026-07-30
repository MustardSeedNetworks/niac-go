package api

import (
	"bytes"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	niacsnmp "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/walkprofile"
)

func (s *Server) writeExistingWalkReview(
	w http.ResponseWriter,
	r *http.Request,
	walkName string,
	requested []byte,
) {
	content, err := s.library.ReadFile(library.KindWalks, walkName)
	if err == nil && !bytes.Equal(content, requested) {
		writeError(
			w, r, http.StatusConflict, "walk_exists",
			"A different captured walk already uses this name", nil,
		)
		return
	}
	if err == nil {
		entries, parseErr := niacsnmp.ParseWalkContent(content)
		if parseErr == nil && len(entries) > 0 {
			s.writeJSON(w, walkprofile.Infer(walkName, entries))
			return
		}
		err = parseErr
	}
	s.logger.ErrorContext(
		r.Context(), "[API] resume captured walk review", "name", walkName, "error", err,
	)
	writeError(
		w, r, http.StatusInternalServerError, "walk_review_failed",
		"Captured walk review could not be resumed", nil,
	)
}
