package protocols

import "github.com/MustardSeedNetworks/niac-go/internal/logging"

// Start begins a single-use protocol stack. A stopped stack cannot be restarted.
func (s *Stack) Start() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	if s.started {
		if s.stopped {
			return ErrStackStopped
		}
		return ErrStackAlreadyRunning
	}
	if err := s.ValidateBehaviorActions(); err != nil {
		return err
	}
	s.started = true
	s.running.Store(true)

	// Start receive thread
	s.wg.Add(1)

	go s.receiveThread()

	// Start decode thread
	s.wg.Add(1)

	go s.decodeThread()

	// Start send thread
	s.wg.Add(1)

	go s.sendThread()

	// Start babble thread (periodic packet generation)
	s.wg.Add(1)

	go s.babbleThread()

	s.wg.Go(func() { s.notifications.Run(s.stopChan) })

	// Start discovery protocol periodic advertisements
	s.lldpHandler.Start()
	s.cdpHandler.Start()
	s.edpHandler.Start()
	s.fdpHandler.Start()
	s.startNeighborCleanupLoop()
	s.startSessionCleanupLoop()
	s.startBehaviorTimelines()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		logging.Debugf("Protocol stack started")
	}

	return nil
}
