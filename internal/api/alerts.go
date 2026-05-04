package api

import (
	"context"
	"encoding/json"
	"fmt"
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

	body := s.buildAlertBody(cfg, total)

	// SSRF defense-in-depth: re-validate URL at send time (config may have been
	// written by an older version, or DNS may have been rebound since validation).
	if err := validateWebhookURLSSRF(cfg.WebhookURL); err != nil {
		s.logger.Error("Alert webhook rejected", "error", err)

		return
	}

	s.postAlertWebhook(cfg.WebhookURL, body)
}

// buildAlertBody serializes the alert payload JSON.
func (s *Server) buildAlertBody(cfg AlertConfig, total uint64) []byte {
	// SECURITY FIX #161: Thread-safe access to Interface
	body, _ := json.Marshal(map[string]any{
		"type":        "packet_threshold",
		"threshold":   cfg.PacketsThreshold,
		"total":       total,
		"interface":   s.currentInterface(),
		"triggeredAt": time.Now().UTC(),
	})

	return body
}

// postAlertWebhook sends the alert payload to the configured webhook URL.
func (s *Server) postAlertWebhook(webhookURL string, body []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeoutSecs*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, webhookURL, strings.NewReader(string(body)),
	)
	if err != nil {
		s.logger.Error("Alert webhook error", "error", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: webhookTimeoutSecs * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // Do not follow redirects to prevent SSRF
		},
	}

	resp, doErr := client.Do(req)
	if doErr != nil {
		s.logger.Error("Alert webhook request failed", "error", doErr)

		return
	}

	_ = resp.Body.Close()
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
