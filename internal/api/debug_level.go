package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Debug-level wire values. These map 1:1 onto the protocol stack's numeric
// debug ladder (protocols.DebugLevel*). The stack reads the global level live
// on every packet (Stack.GetDebugLevel), so a PUT here takes effect immediately
// with no restart and no per-handler propagation.
const (
	debugLevelOff     = "off"
	debugLevelBasic   = "basic"
	debugLevelInfo    = "info"
	debugLevelVerbose = "verbose"
	debugLevelTrace   = "trace"
)

// defaultDebugLevelInt mirrors daemon.DefaultDebugLevel (DebugLevelBasic). The
// api package cannot import daemon (daemon imports api), so it is anchored to
// the same stack constant daemon derives its default from; TestHandleDebugLevelGet
// pins the "basic" wire form.
const defaultDebugLevelInt = protocols.DebugLevelBasic

// debugLevelResponse is returned by GET and PUT /api/v1/debug/level.
type debugLevelResponse struct {
	Level        string `json:"level"`
	DefaultLevel string `json:"defaultLevel"`
}

// updateDebugLevelRequest is the PUT body for /api/v1/debug/level.
type updateDebugLevelRequest struct {
	Level string `json:"level"`
}

// debugLevelToString converts a stack debug int to its wire string, clamping
// out-of-range values into the known ladder so the UI never sees an unknown
// token.
func debugLevelToString(level int) string {
	switch {
	case level <= 0:
		return debugLevelOff
	case level == protocols.DebugLevelBasic:
		return debugLevelBasic
	case level == protocols.DebugLevelInfo:
		return debugLevelInfo
	case level == protocols.DebugLevelVerbose:
		return debugLevelVerbose
	default: // >= DebugLevelTrace
		return debugLevelTrace
	}
}

// debugLevelFromString converts a wire string to a stack debug int. ok is false
// for any value outside the known ladder, so the handler rejects it with 400.
func debugLevelFromString(s string) (int, bool) {
	switch s {
	case debugLevelOff:
		return 0, true
	case debugLevelBasic:
		return protocols.DebugLevelBasic, true
	case debugLevelInfo:
		return protocols.DebugLevelInfo, true
	case debugLevelVerbose:
		return protocols.DebugLevelVerbose, true
	case debugLevelTrace:
		return protocols.DebugLevelTrace, true
	default:
		return 0, false
	}
}

// handleDebugLevel serves GET/PUT /api/v1/debug/level — the single global debug
// verbosity control. GET reports the current global level plus the boot
// default; PUT sets the global level, which every protocol handler honors live
// via Stack.GetDebugLevel(). There is no per-protocol surface: NIAC's handlers
// read one global level, so one control is the honest contract.
func (s *Server) handleDebugLevel(w http.ResponseWriter, r *http.Request) {
	stack := s.currentStack()
	if stack == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_active_simulation",
			"Debug level is unavailable until a simulation is running", nil)

		return
	}

	debugConfig := stack.GetDebugConfig()

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, debugLevelResponse{
			Level:        debugLevelToString(debugConfig.GetGlobal()),
			DefaultLevel: debugLevelToString(defaultDebugLevelInt),
		})
	case http.MethodPut:
		var req updateDebugLevelRequest
		if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
			return
		}

		level, ok := debugLevelFromString(req.Level)
		if !ok {
			writeError(w, r, http.StatusBadRequest, "invalid_level",
				"level must be one of: off, basic, info, verbose, trace", nil)

			return
		}

		debugConfig.SetGlobal(level)
		s.writeJSON(w, debugLevelResponse{
			Level:        debugLevelToString(level),
			DefaultLevel: debugLevelToString(defaultDebugLevelInt),
		})
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
