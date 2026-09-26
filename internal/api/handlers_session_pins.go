package api

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// ErrAttachmentPoolRequired means the session's attachment is not a port pool,
// so there is no port to pin a client to.
var ErrAttachmentPoolRequired = errors.New("the session's attachment is not a port pool")

// AttachmentPin moves one observed client to one port of the session's
// attachment pool.
type AttachmentPin struct {
	MAC       string `json:"mac"`
	Device    string `json:"device"`
	Interface string `json:"interface"`
}

// handleSessionPins re-pins a client. Until the running stack can reload in
// place (AP-6) the move is a restart of this one session, so every client of
// the session is placed afresh; sibling sessions are not touched.
func (s *Server) handleSessionPins(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}
	if s.daemon == nil {
		writeError(w, r, http.StatusNotImplemented, "daemon_required",
			"Daemon mode is required", nil)
		return
	}
	var req AttachmentPin
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	mac, err := net.ParseMAC(req.MAC)
	if err != nil || len(mac) != 6 {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed",
			[]ErrorDetail{{Field: "mac", Issue: "must be a 48-bit MAC address"}})
		return
	}
	pin := AttachmentPin{
		MAC:       mac.String(),
		Device:    strings.TrimSpace(req.Device),
		Interface: strings.TrimSpace(req.Interface),
	}
	if pin.Device == "" || pin.Interface == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed",
			[]ErrorDetail{{Field: "port", Issue: "device and interface are required"}})
		return
	}
	if err = s.daemon.PinAttachmentClient(session.id, pin); err != nil {
		if errors.Is(err, ErrAttachmentPoolRequired) {
			writeError(w, r, http.StatusConflict, "attachment_pool_required",
				"This scenario's attachment is not a port pool", nil)
			return
		}
		s.handleSimulationStartError(w, r, err)
		return
	}
	s.writeJSON(w, pin)
}
