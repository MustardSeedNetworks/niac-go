package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.getAlertConfig())
	case http.MethodPut, http.MethodPost:
		var req AlertConfig
		if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
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

			s.evaluateAlertThreshold(
				stats.PacketsSent+stats.PacketsReceived,
				cfg.PacketsThreshold,
				time.Now(),
			)
		case <-stop:
			return
		}
	}
}

// evaluateAlertThreshold fires at most one alert per crossing, then at most
// one per cooldown while the total stays across the threshold.
//
// The previous rule was "fire whenever the total differs from the last one
// alerted". The packet total is cumulative, so on a running simulation it
// differs on every tick -- one crossing sent a webhook every five seconds for
// as long as the session lived. The receiver of that storm was whatever
// operators had configured, which is why this is a defect rather than noise.
func (s *Server) evaluateAlertThreshold(total, threshold uint64, now time.Time) {
	s.alertMu.Lock()
	fire, firing := shouldFireAlert(s.alertFiring, s.lastAlertAt, total, threshold, now)
	s.alertFiring = firing
	if fire {
		s.lastAlert = total
		s.lastAlertAt = now
	}
	s.alertMu.Unlock()

	if fire {
		go s.sendAlert(total)
	}
}

// shouldFireAlert decides whether a tick notifies, and what the firing state
// becomes. Kept pure so the decision is testable without a webhook: the
// send path refuses loopback by SSRF policy, so a test receiver can never
// stand in for the real one.
//
// It returns whether this tick notifies, and the firing state to carry into
// the next one: true at most once per crossing, then at most once per cooldown
// while the total stays across the threshold.
func shouldFireAlert(
	firing bool,
	lastAlertAt time.Time,
	total, threshold uint64,
	now time.Time,
) (bool, bool) {
	if total < threshold {
		// Below the line again: re-arm so the next crossing is reported.
		return false, false
	}

	if firing && now.Sub(lastAlertAt) < alertCooldown {
		return false, true
	}

	return true, true
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
	// SSRF defence runs first so CodeQL sees the URL is bounded before any
	// downstream parse/dispatch — the function rejects blocked hostnames and
	// private/loopback IP literals.
	if ssrfErr := validateWebhookURLSSRF(webhookURL); ssrfErr != nil {
		s.logger.Error("Alert webhook URL rejected", "error", ssrfErr)
		return
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		s.logger.Error("Invalid alert webhook URL", "error", err)
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		s.logger.Error("Alert webhook URL must use http or https scheme")
		return
	}

	// Inline SSRF barrier at the dispatch site. validateWebhookURLSSRF above
	// performs the same checks, but CodeQL's go/request-forgery query needs
	// to see the host check directly at the http.Client.Do call.
	host := parsedURL.Hostname()
	if host == "" {
		s.logger.Error("Alert webhook URL has empty host")
		return
	}
	// Admin-side hostname allowlist. When configured, only exact hostname
	// matches are accepted — this is the canonical CodeQL barrier for
	// go/request-forgery and lets a deployment opt out of accepting
	// arbitrary user-supplied webhook destinations entirely.
	if allowed := s.cfg.WebhookAllowedHosts; len(allowed) > 0 {
		if !slices.Contains(allowed, host) {
			s.logger.Error("Alert webhook host not in allowlist", "host", host)
			return
		}
	} else if ip := net.ParseIP(host); ip != nil {
		// No allowlist configured — fall back to the IP-class filter so a
		// raw-IP literal can't reach private/loopback/link-local space.
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
			ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
			s.logger.Error("Alert webhook URL targets a non-routable address", "host", host)
			return
		}
	}

	// Reconstruct the dispatched URL from validated components so the value
	// passed to http.NewRequestWithContext is a single bounded source.
	sanitizedURL := (&url.URL{
		Scheme:   parsedURL.Scheme,
		Host:     parsedURL.Host,
		Path:     parsedURL.Path,
		RawQuery: parsedURL.RawQuery,
	}).String()

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeoutSecs*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, sanitizedURL, strings.NewReader(string(body)),
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
	s.alertFiring = false
	s.lastAlertAt = time.Time{}

	// Start new alert loop if threshold is set
	if cfg.PacketsThreshold > 0 {
		stopChan := make(chan struct{})
		s.alertStop = stopChan
		// Start goroutine while still holding lock to prevent race
		go s.alertLoop(stopChan)
	}
}
