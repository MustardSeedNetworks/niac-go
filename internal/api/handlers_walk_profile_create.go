package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	niacsnmp "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	"github.com/MustardSeedNetworks/niac-go/internal/walkprofile"
)

type createCapturedProfileRequest struct {
	Role       string `json:"role"`
	DeviceType string `json:"deviceType"`
	Vendor     string `json:"vendor"`
	Model      string `json:"model"`
	Platform   string `json:"platform"`
	Software   string `json:"software"`
	WalkName   string `json:"walkName"`
}

func (s *Server) handleCapturedProfileCreate(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	var request createCapturedProfileRequest
	if !decodeJSONStrict(w, r, &request, MaxRequestBodySize) {
		return
	}
	if !validCapturedWalkName(request.WalkName) {
		writeError(w, r, http.StatusBadRequest, "invalid_walk", "Captured walk is unavailable", nil)
		return
	}
	entry, err := s.library.FileEntryByName(library.KindWalks, request.WalkName)
	if err != nil || entry.SizeBytes > MaxWalkImportSize {
		writeError(w, r, http.StatusBadRequest, "invalid_walk", "Captured walk is unavailable", nil)
		return
	}
	content, err := s.library.ReadFile(library.KindWalks, request.WalkName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, library.ErrNotFound) || errors.Is(err, library.ErrInvalidName) {
			status = http.StatusBadRequest
		}
		writeError(w, r, status, "invalid_walk", "Captured walk is unavailable", nil)
		return
	}
	entries, err := niacsnmp.ParseWalkContent(content)
	if err != nil || len(entries) == 0 {
		writeError(
			w, r, http.StatusUnprocessableEntity, "invalid_walk",
			"Captured walk contains no reusable SNMP data", nil,
		)
		return
	}
	profile := walkprofile.Infer(request.WalkName, entries).Profile
	profile.Role = strings.TrimSpace(request.Role)
	profile.DeviceType = strings.TrimSpace(request.DeviceType)
	profile.Vendor = strings.TrimSpace(request.Vendor)
	profile.Model = strings.TrimSpace(request.Model)
	profile.Platform = strings.TrimSpace(request.Platform)
	profile.Software = strings.TrimSpace(request.Software)
	if err = scenario.SaveCustomProfile(s.library.Root(), profile); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, scenario.ErrInvalidProfile) {
			status = http.StatusBadRequest
		} else if errors.Is(err, scenario.ErrProfileExists) {
			status = http.StatusConflict
		}
		message := "Captured profile could not be saved"
		if status != http.StatusInternalServerError {
			message = err.Error()
		}
		writeError(w, r, status, "profile_save_failed", message, nil)
		return
	}
	s.logger.InfoContext(
		r.Context(), "[API] captured scenario profile created",
		"role", profile.Role, "walk", profile.WalkName,
	)
	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, profile)
}
