package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

const deviceActionIDBytes = 16

type deviceActionRequest struct {
	Device string `json:"device"`
	Action string `json:"action"`
}

// handleDeviceAction runs one operation against one device, now, the way the
// fault routes arm one condition. Before this existed the only caller of
// Stack.ExecuteDeviceAction was the behavior timeline runner, so a reboot or a
// spanning-tree topology change could be scheduled and waited for but never
// triggered by hand -- two problems from the NetAlly cross-reference that every
// other problem's screen could not reach.
func (s *Server) handleDeviceAction(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_simulation",
			"No simulation running", nil)
		return
	}

	var request deviceActionRequest
	if !decodeJSONStrict(w, r, &request, MaxRequestBodySize) {
		return
	}
	if request.Device == "" || request.Action == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"device and action are required", nil)
		return
	}

	// Each request is a fresh intent, so it carries a fresh identity. The
	// store's identity map exists to keep a replayed timeline phase from
	// firing twice, not to collapse two deliberate operator actions into one.
	id, err := newDeviceActionID()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] device action identity", "error", err)
		writeError(w, r, http.StatusInternalServerError, "action_failed",
			"Device action could not be started", nil)
		return
	}

	if err = stack.ExecuteDeviceAction(
		request.Device, devicestate.DeviceActionType(request.Action), id,
	); err != nil {
		status, code := deviceActionErrorStatus(err)
		writeError(w, r, status, code, err.Error(), nil)
		return
	}

	s.writeJSON(w, map[string]string{
		"device": request.Device, "action": request.Action, "status": "executed",
	})
}

func deviceActionErrorStatus(err error) (int, string) {
	switch {
	case errors.Is(err, devicestate.ErrDeviceActionInvalid):
		return http.StatusBadRequest, "unknown_action"
	case errors.Is(err, protocols.ErrDeviceActionUnobservable):
		// The device cannot publish the effect, so running it would report
		// success while nothing anywhere changed.
		return http.StatusConflict, "action_unobservable"
	case errors.Is(err, devicestate.ErrDeviceActionLimit):
		return http.StatusTooManyRequests, "action_limit"
	default:
		return http.StatusNotFound, "device_not_found"
	}
}

func newDeviceActionID() (string, error) {
	var bytes [deviceActionIDBytes]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
