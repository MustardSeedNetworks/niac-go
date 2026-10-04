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

// ErrAttachmentPortOccupied means another client is plugged into the port a
// re-pin names. A move never unplugs someone else.
var ErrAttachmentPortOccupied = errors.New("another client is plugged into that port")

// ErrAttachmentPortShut means the port a re-pin names is administratively shut,
// so a client moved there would get no link.
var ErrAttachmentPortShut = errors.New("that port is administratively shut")

// AttachmentPin moves one observed client to one port of the session's
// attachment pool.
type AttachmentPin struct {
	MAC       string `json:"mac"`
	Device    string `json:"device"`
	Interface string `json:"interface"`
}

// handleSessionPins re-pins a client on the running session. Only that client
// moves; the session, its binding and every other client stay as they are.
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
		if errors.Is(err, ErrAttachmentPortOccupied) {
			writeError(w, r, http.StatusConflict, "attachment_port_occupied",
				"Another client is plugged into that port", nil)
			return
		}
		if errors.Is(err, ErrAttachmentPortShut) {
			writeError(w, r, http.StatusConflict, "attachment_port_shut",
				"That port is administratively shut", nil)
			return
		}
		s.handleSimulationStartError(w, r, err)
		return
	}
	s.writeJSON(w, pin)
}
