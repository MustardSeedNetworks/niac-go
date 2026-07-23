package devicecli

import (
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *Session) executePrivileged(fields []string) Response {
	if len(fields) == 2 && fields[0] == "checkpoint" {
		if !validHostname(fields[1]) {
			return Response{Output: "% Invalid checkpoint name"}
		}
		s.state.SaveCheckpoint(fields[1])
		return Response{Output: "[OK]"}
	}
	if len(fields) == 3 && fields[0] == "rollback" && fields[1] == "checkpoint" {
		if err := s.state.RestoreCheckpoint(fields[2]); err != nil {
			if errors.Is(err, devicestate.ErrCheckpointNotFound) {
				return Response{Output: "% Checkpoint not found"}
			}
			return Response{Output: "% Rollback failed"}
		}
		return Response{Output: "[OK]"}
	}
	command := strings.Join(fields, " ")
	if response, handled := s.executeLifecycle(command); handled {
		return response
	}
	switch command {
	case "disable":
		s.mode = ModeUser
	case "configure terminal", "conf t":
		s.mode = ModeGlobalConfig
		return Response{Output: "Enter configuration commands, one per line."}
	case commandExit, "logout":
		return Response{Close: true}
	default:
		return Response{Output: invalidInput}
	}

	return Response{}
}
