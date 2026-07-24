package protocols

import "time"

func (s *Stack) startSessionCleanupLoop() {
	if s.tcpHandler == nil || s.tcpHandler.ssh == nil {
		return
	}

	s.wg.Go(func() {
		ticker := time.NewTicker(sshSessionCleanup)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.tcpHandler.ssh.cleanupExpired()
			case <-s.stopChan:
				return
			}
		}
	})
}
