package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.getAlertConfig())
	case http.MethodPut, http.MethodPost:
		// SECURITY FIX MEDIUM-3: Add request body size limit
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		var req AlertConfig
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			if err.Error() == ErrMsgRequestBodyTooLarge {
				writeError(
					w,
					r,
					http.StatusRequestEntityTooLarge,
					"request_too_large",
					fmt.Sprintf(
						"Request body exceeds maximum size of %d bytes",
						MaxRequestBodySize,
					),
					nil,
				)

				return
			}

			writeError(w, r, http.StatusBadRequest, "invalid_request",
				"Failed to parse request body", nil)

			return
		}

		// SECURITY FIX MEDIUM-3: Validate input fields
		if validationErrors := validateAlertConfig(req); len(validationErrors) > 0 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Alert configuration validation failed", validationErrors)

			return
		}

		s.updateAlertConfig(req)
		s.writeJSON(w, s.getAlertConfig())
	default:
		w.Header().Set("Allow", "GET, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getAlertConfig() AlertConfig {
	s.alertMu.RLock()
	defer s.alertMu.RUnlock()

	return s.cfg.Alert
}

func (s *Server) updateAlertConfig(cfg AlertConfig) {
	// SECURITY FIX #2.8.1: Prevent race condition on alertStop channel
	s.alertMu.Lock()
	defer s.alertMu.Unlock()

	// Stop existing alert loop if running
	if s.alertStop != nil {
		// Check if channel is already closed to prevent panic
		select {
		case <-s.alertStop:
			// Already closed
		default:
			// Safe to close
			close(s.alertStop)
		}

		s.alertStop = nil
	}

	s.cfg.Alert = cfg
	s.lastAlert = 0

	// Start new alert loop if threshold is set
	if cfg.PacketsThreshold > 0 {
		stopChan := make(chan struct{})
		s.alertStop = stopChan
		// Start goroutine while still holding lock to prevent race
		go s.alertLoop(stopChan)
	}
}

func (s *Server) alertLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(alertTickerSecs * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cfg := s.getAlertConfig()
			if cfg.PacketsThreshold == 0 {
				continue
			}

			s.configMu.RLock()
			stack := s.cfg.Stack
			s.configMu.RUnlock()

			if stack == nil {
				continue
			}

			stats := stack.GetStats()

			total := stats.PacketsSent + stats.PacketsReceived
			if total >= cfg.PacketsThreshold {
				s.alertMu.Lock()

				if total != s.lastAlert {
					s.lastAlert = total
					go s.sendAlert(total)
				}

				s.alertMu.Unlock()
			}
		case <-stop:
			return
		}
	}
}

func (s *Server) sendAlert(total uint64) {
	s.logger.Warn("Alert: packet threshold exceeded", "total", total)

	cfg := s.getAlertConfig()
	if cfg.WebhookURL == "" {
		return
	}

	// SECURITY FIX #161: Thread-safe access to Interface
	body, _ := json.Marshal(map[string]any{
		"type":        "packet_threshold",
		"threshold":   cfg.PacketsThreshold,
		"total":       total,
		"interface":   s.currentInterface(),
		"triggeredAt": time.Now().UTC(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeoutSecs*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		cfg.WebhookURL,
		strings.NewReader(string(body)),
	)
	if err != nil {
		s.logger.Error("Alert webhook error", "error", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")

	// FIX #314, #315: Custom transport to validate resolved IPs and deny redirects
	dialer := &net.Dialer{Timeout: webhookTimeoutSecs * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, fmt.Errorf("invalid address: %s", addr)
			}

			addrs, resolveErr := net.DefaultResolver.LookupIPAddr(ctx, host)
			if resolveErr != nil {
				return nil, fmt.Errorf("cannot resolve host: %s", host)
			}

			for _, addr := range addrs {
				if blockedErr := isBlockedIP(addr.IP); blockedErr != nil {
					return nil, fmt.Errorf("blocked destination: %w", blockedErr)
				}
			}

			// Connect to first allowed IP
			return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
		},
	}

	client := &http.Client{
		Timeout:   webhookTimeoutSecs * time.Second,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("redirects not allowed for webhook URLs")
		},
	}

	resp, doErr := client.Do(req)
	if doErr != nil {
		s.logger.Error("Alert webhook request failed", "error", doErr)

		return
	}

	_ = resp.Body.Close()
}
