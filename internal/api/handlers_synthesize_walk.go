package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/library"
	"github.com/krisarmstrong/niac-go/internal/protocols/snmp/synth"
)

// SynthesizeWalkRequest is the body for POST
// /api/v1/devices/{hostname}/synthesize-walk. See
// docs/design/2026-05-baseline-walk-synthesis.md for the contract.
//
// When Model is non-empty, the daemon resolves the per-model
// profile (synth.PickModel) — the walk gets the model's real
// sysObjectID + sysDescr. Vendor still required because Model
// strings aren't globally unique across vendors.
//
// When Model is empty, falls back to synth.Pick(vendor, deviceType)
// — the original v0.66.x behaviour with the per-vendor generic
// profile.
type SynthesizeWalkRequest struct {
	Vendor         string `json:"vendor"`
	Model          string `json:"model,omitempty"`
	InterfaceCount int    `json:"interfaceCount,omitempty"`
	Community      string `json:"community,omitempty"`
}

// SynthesizeWalkResponse is the 201 body. WalkPath is library-relative
// so the device's walk_file can be set directly to this string.
type SynthesizeWalkResponse struct {
	WalkPath           string `json:"walkPath"`
	OIDCount           int    `json:"oidCount"`
	SizeBytes          int    `json:"sizeBytes"`
	PreviousWalkBacked string `json:"previousWalkBacked,omitempty"`
}

// handleSynthesizeWalkModels handles GET /api/v1/synthesize-walk/models
// — returns every defined (vendor, model, type) triple so the UI can
// build the cascading Generate-walk picker without a per-vendor
// round-trip. Read-only; the response body is the same list returned
// by synth.AllModels() so callers can group/sort client-side.
func (s *Server) handleSynthesizeWalkModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}
	s.writeJSON(w, synth.AllModels())
}

// dispatchDeviceSubpath routes /api/v1/devices/{hostname}/{action}
// to the right per-action handler. Today there's exactly one action
// (synthesize-walk); the dispatch shape leaves room for adding more
// per-device actions (clone-walk, regenerate-mac, etc.) without
// touching the router every time.
func (s *Server) dispatchDeviceSubpath(w http.ResponseWriter, r *http.Request) {
	const hostnameAndAction = 2 // {hostname}/{action} after the prefix strip
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/")
	parts := strings.SplitN(rest, "/", hostnameAndAction)
	if len(parts) != hostnameAndAction || parts[1] == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "Unknown device action", nil)
		return
	}
	action := parts[1]
	switch action {
	case "synthesize-walk":
		s.handleSynthesizeWalk(w, r)
	default:
		writeError(w, r, http.StatusNotFound, "not_found",
			fmt.Sprintf("Unknown device action: %s", action), nil)
	}
}

// handleSynthesizeWalk handles POST
// /api/v1/devices/{hostname}/synthesize-walk. Generates a baseline
// SNMP walk file from the device's type + a vendor selector, writes
// it to <library>/walks/synthesized/{hostname}.walk, and updates the
// device's walk_file field to point at it.
//
// Validation order (matches design-doc resolutions):
//  1. library must be open (else 503)
//  2. method must be POST (else 405)
//  3. hostname valid name (else 400)
//  4. JSON body decodes (else 400)
//  5. device exists in current config (else 404)
//  6. (vendor, device.type) is synthesizable (else 422 — no silent
//     fallback per resolution #2)
//  7. write + attach (any failure → 500)
func (s *Server) handleSynthesizeWalk(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	hostname := extractHostnameFromSynthesizePath(r.URL.Path)
	if hostname == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Hostname required", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	var req SynthesizeWalkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}

	dev := findDeviceByName(s.cfg.Config, hostname)
	if dev == nil {
		writeError(w, r, http.StatusNotFound, "not_found",
			fmt.Sprintf("Device not found: %s", hostname), nil)
		return
	}

	vendor := synth.Vendor(strings.TrimSpace(req.Vendor))
	if vendor == "" {
		vendor = synth.VendorGeneric
	}
	deviceType := synth.DeviceType(strings.ToLower(strings.TrimSpace(dev.Type)))
	model := synth.Model(strings.TrimSpace(req.Model))

	// Prefer the per-model profile when the request specified one —
	// the synthesised walk gets the real platform sysDescr + OID.
	// Empty model falls back to the per-vendor generic.
	var (
		profile    synth.Profile
		profileErr error
	)
	if model != "" {
		profile, profileErr = synth.PickModel(vendor, model)
	} else {
		profile, profileErr = synth.Pick(vendor, deviceType)
	}
	if profileErr != nil {
		// Per resolution #2 — no silent fallback. The daemon returns
		// 422 so the UI can show "Vendor X has no profile for type Y".
		// The UI side disables invalid combinations in the dropdown,
		// so this is the safety-net path for stale or scripted callers.
		writeError(w, r, http.StatusUnprocessableEntity, "unsupported_combo",
			profileErr.Error(), nil)
		return
	}

	input := synth.DeviceInput{
		Hostname:      dev.Name,
		IP:            primaryIPString(dev),
		AdditionalIPs: additionalIPStrings(dev),
		Community:     req.Community,
	}
	walkBytes := synth.Build(profile, input, synth.BuildOptions{
		InterfaceCount: req.InterfaceCount,
	})

	relPath := "synthesized/" + dev.Name + ".walk"
	// Was there a prior file? Check before WriteFile, since after
	// WriteFile any prior copy is at relPath+".bak".
	hadPrior := s.libraryFileExists(library.KindWalks, relPath)

	if writeErr := s.library.WriteFile(library.KindWalks, relPath, walkBytes, true); writeErr != nil {
		s.logger.Error("[API] synthesize-walk: write failed", "device", hostname, "error", writeErr)
		writeError(w, r, http.StatusInternalServerError, "write_failed",
			"Failed to write synthesised walk", nil)
		return
	}

	// Update the device's walk_file and save the config. The walk path
	// is library-relative; the walk validator falls back to the
	// library/walks/ dir (per #558) so this string Just Works as
	// device.snmp_config.walk_file.
	updateErr := s.attachWalkToDevice(dev.Name, relPath)
	if updateErr != nil {
		s.logger.Error("[API] synthesize-walk: attach failed", "device", hostname, "error", updateErr)
		writeError(w, r, http.StatusInternalServerError, "attach_failed",
			"Walk synthesised but could not attach to device", nil)
		return
	}

	resp := SynthesizeWalkResponse{
		WalkPath:  relPath,
		OIDCount:  countOIDLines(walkBytes),
		SizeBytes: len(walkBytes),
	}
	if hadPrior {
		resp.PreviousWalkBacked = relPath + ".bak"
	}
	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, resp)
}

// extractHostnameFromSynthesizePath returns the {hostname} segment
// from /api/v1/devices/{hostname}/synthesize-walk. Returns "" if the
// path doesn't match the expected shape.
func extractHostnameFromSynthesizePath(path string) string {
	rest := strings.TrimPrefix(path, "/api/v1/devices/")
	rest = strings.TrimSuffix(rest, "/synthesize-walk")
	if strings.Contains(rest, "/") {
		// Malformed — hostname can't contain a slash.
		return ""
	}
	return rest
}

// findDeviceByName scans the loaded config for a device matching the
// hostname. Returns nil when not found. Uses the same comparison as
// findDeviceIndex elsewhere in this package (case-sensitive exact
// match — hostnames in niac configs are canonical lowercase).
func findDeviceByName(cfg *config.Config, hostname string) *config.Device {
	if cfg == nil {
		return nil
	}
	for i := range cfg.Devices {
		if cfg.Devices[i].Name == hostname {
			return &cfg.Devices[i]
		}
	}
	return nil
}

// primaryIPString returns the device's first IP as a dotted string,
// or "" when the device has no IPs (e.g. a placeholder host that's
// only there for trunk-port references).
func primaryIPString(dev *config.Device) string {
	if dev == nil || len(dev.IPAddresses) == 0 {
		return ""
	}
	return dev.IPAddresses[0].String()
}

// additionalIPStrings returns every IP after the first one.
func additionalIPStrings(dev *config.Device) []string {
	if dev == nil || len(dev.IPAddresses) < 2 {
		return nil
	}
	out := make([]string, 0, len(dev.IPAddresses)-1)
	for _, ip := range dev.IPAddresses[1:] {
		out = append(out, ip.String())
	}
	return out
}

// libraryFileExists returns true when {kind}/{relPath} is on disk.
// Used to decide whether the response should advertise a .bak file.
func (s *Server) libraryFileExists(kind library.Kind, relPath string) bool {
	if s.library == nil {
		return false
	}
	abs := filepath.Join(s.library.SubDir(kind), filepath.FromSlash(relPath))
	_, err := os.Stat(abs)
	return err == nil
}

// attachWalkToDevice rewrites the device's walk_file to point at
// relPath (library-relative) and persists the config. Returns
// without re-saving if the walk_file is already set to the same
// value (idempotent — matches the resolution #4 stance that
// retry-storm protection lives on the UI debounce, not the server).
func (s *Server) attachWalkToDevice(hostname, relPath string) error {
	if s.cfg.Config == nil {
		return errors.New("config not loaded")
	}
	newCfg := *deepCopyConfig(s.cfg.Config)
	idx := findDeviceIndex(newCfg.Devices, hostname)
	if idx < 0 {
		return fmt.Errorf("device not found in current config: %s", hostname)
	}
	if newCfg.Devices[idx].SNMPConfig.WalkFile == relPath {
		return nil // no-op rewrite
	}
	newCfg.Devices[idx].SNMPConfig.WalkFile = relPath
	if newCfg.Devices[idx].SNMPConfig.Community == "" {
		newCfg.Devices[idx].SNMPConfig.Community = "public"
	}
	return s.saveConfig(&newCfg)
}

// countOIDLines counts non-comment, non-blank lines in the walk body
// — i.e. the number of OIDs the synth pass emitted. Useful for the
// 201 response so the UI can show "Generated walk with N OIDs".
func countOIDLines(walk []byte) int {
	count := 0
	for _, line := range strings.Split(string(walk), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		count++
	}
	return count
}
