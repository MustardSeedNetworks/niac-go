package api

import (
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	niacsnmp "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/sanitize"
	"github.com/MustardSeedNetworks/niac-go/internal/walkcapture"
	"github.com/MustardSeedNetworks/niac-go/internal/walkprofile"
)

var importedWalkNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,126}\.walk$`)

const captureWriteDeadline = 65 * time.Second

type walkImportRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type walkCaptureRequest struct {
	Name    string              `json:"name"`
	Capture walkcapture.Request `json:"capture"`
}

func (s *Server) handleWalkImport(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	var request walkImportRequest
	if !decodeJSONStrict(w, r, &request, MaxWalkImportBodySize) {
		return
	}
	if len(request.Content) == 0 || len(request.Content) > MaxWalkImportSize {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_walk",
			"Walk content must be between 1 byte and 16 MiB",
			nil,
		)
		return
	}
	s.createWalkReview(w, r, request.Name, []byte(request.Content))
}

func (s *Server) handleWalkCaptureProfile(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	var request walkCaptureRequest
	if !decodeJSONStrict(w, r, &request, MaxRequestBodySize) {
		return
	}
	if err := http.NewResponseController(w).SetWriteDeadline(
		time.Now().Add(captureWriteDeadline),
	); err != nil && !errors.Is(err, http.ErrNotSupported) {
		s.logger.WarnContext(r.Context(), "[API] extend walk capture response deadline", "error", err)
	}
	content, err := walkcapture.Capture(r.Context(), request.Capture)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, walkcapture.ErrInvalidRequest) {
			status = http.StatusBadRequest
		}
		message := "SNMP walk capture failed"
		if status == http.StatusBadRequest {
			message = err.Error()
		}
		writeError(w, r, status, "walk_capture_failed", message, nil)
		return
	}
	s.createWalkReview(w, r, request.Name, content)
}

func (s *Server) createWalkReview(
	w http.ResponseWriter,
	r *http.Request,
	rawName string,
	content []byte,
) {
	name := strings.TrimSpace(rawName)
	if !importedWalkNameRE.MatchString(name) || filepath.Base(name) != name {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_name",
			"Walk name must be a plain .walk filename",
			nil,
		)
		return
	}
	sanitized, _, err := sanitize.Content(content, nil, sanitize.DefaultOptions())
	if err != nil {
		writeError(
			w,
			r,
			http.StatusUnprocessableEntity,
			"sanitize_failed",
			"Walk content could not be sanitized",
			nil,
		)
		return
	}
	sanitized = niacsnmp.NormalizeKnownWalkOIDs(sanitized)
	validation, err := niacsnmp.ValidateWalkContent(name, sanitized)
	if err != nil || !validation.Valid || validation.ValidLines == 0 {
		writeError(
			w,
			r,
			http.StatusUnprocessableEntity,
			"invalid_walk",
			"Walk content is not valid net-snmp output",
			nil,
		)
		return
	}
	entries, err := niacsnmp.ParseWalkContent(sanitized)
	if err != nil || len(entries) == 0 {
		writeError(
			w,
			r,
			http.StatusUnprocessableEntity,
			"invalid_walk",
			"Walk content contains no reusable SNMP data",
			nil,
		)
		return
	}
	walkName := "captured/" + name
	if err = s.library.WriteFileNew(library.KindWalks, walkName, sanitized); errors.Is(
		err,
		library.ErrAlreadyExists,
	) {
		s.writeExistingWalkReview(w, r, walkName, sanitized)
		return
	} else if err != nil {
		s.logger.ErrorContext(
			r.Context(),
			"[API] save sanitized walk capture",
			"name",
			walkName,
			"error",
			err,
		)
		writeError(
			w,
			r,
			http.StatusInternalServerError,
			"walk_save_failed",
			"Sanitized walk could not be saved",
			nil,
		)
		return
	}
	s.writeJSON(w, walkprofile.Infer(walkName, entries))
}

func validCapturedWalkName(name string) bool {
	const prefix = "captured/"
	return strings.HasPrefix(name, prefix) &&
		importedWalkNameRE.MatchString(strings.TrimPrefix(name, prefix)) &&
		filepath.ToSlash(name) == name
}
