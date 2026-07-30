package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
)

func (s *Server) handleBehaviors(w http.ResponseWriter, _ *http.Request) {
	stack := s.currentStack()
	if stack == nil {
		s.writeJSON(w, behavior.Status{State: "idle"})
		return
	}
	s.writeJSON(w, stack.BehaviorStatus())
}
