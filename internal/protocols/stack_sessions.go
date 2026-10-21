package protocols

import (
	"context"
	"time"
)

func (s *Stack) sessionCleanupLoop(ctx context.Context) error {
	ticker := time.NewTicker(sshSessionCleanup)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tcpHandler.ssh.cleanupExpired()
		case <-ctx.Done():
			return nil
		}
	}
}
