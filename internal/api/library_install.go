package api

import (
	"bytes"
	"encoding/base64"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// LibraryInstallRequest is the body for POST /api/v1/library/install: a
// gzip-tar content bundle (same format as `niac content install --bundle`
// and the embedded/deb starter bundles), base64-encoded so it travels as a
// normal JSON body — mirrors PcapUploadRequest's shape for consistency
// across the API's binary-upload surfaces.
type LibraryInstallRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"` // Base64 encoded gzip-tar bundle
}

// LibraryInstallResponse reports what the bundle installed, projected from
// content.Manifest. PerKind keys are library.Kind values ("networks",
// "walks", "pcaps").
type LibraryInstallResponse struct {
	Success     bool                 `json:"success"`
	Files       int                  `json:"files"`
	Directories int                  `json:"directories"`
	Bytes       int64                `json:"bytes"`
	PerKind     map[library.Kind]int `json:"perKind"`
	Message     string               `json:"message"`
}

// handleLibraryInstall handles POST /api/v1/library/install: the UI-upload
// counterpart to `niac content install --bundle` (internal/content/extract.go
// header). This replaces library content wholesale, so it is registered as
// admin-class (admin scope + CSRF + write rate limit) alongside
// /api/v1/config/import — the other whole-store replace endpoint.
func (s *Server) handleLibraryInstall(w http.ResponseWriter, r *http.Request) {
	// Method gating (POST-only) is enforced declaratively by the route registry.
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}

	var req LibraryInstallRequest
	if !decodeJSONStrict(w, r, &req, MaxLibraryInstallBodySize) {
		return
	}
	if req.Data == "" {
		writeError(w, r, http.StatusBadRequest, "missing_data", "bundle data is required", nil)
		return
	}

	bundle, decodeErr := base64.StdEncoding.DecodeString(req.Data)
	if decodeErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_base64", "Invalid base64 encoded data", nil)
		return
	}

	manifest, extractErr := content.Extract(bytes.NewReader(bundle), s.library.Root(), content.ExtractOptions{})
	if extractErr != nil {
		s.logger.WarnContext(r.Context(), "[API] library install: bundle rejected",
			"filename", req.Filename, "error", extractErr)
		writeError(w, r, http.StatusBadRequest, "bundle_invalid", extractErr.Error(), nil)
		return
	}

	s.writeJSON(w, LibraryInstallResponse{
		Success:     true,
		Files:       manifest.Files,
		Directories: manifest.Directories,
		Bytes:       manifest.Bytes,
		PerKind:     manifest.PerKind,
		Message:     "Content bundle installed",
	})
}
