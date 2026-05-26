package api

import (
	"net/http"
	"time"
)

// registerAPIRoutes registers all API endpoints on the provided mux.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	// Public version endpoint for deployment validation (no auth required)
	// Intentionally placed before authenticated routes for clarity
	mux.HandleFunc("/__version", s.recoverMiddleware(s.handleBuildVersion))

	// SECURITY FIX #2.8.1: Wrap all handlers with panic recovery middleware
	// SECURITY FIX LOW-1: CSRF token endpoint for clients to retrieve token
	mux.HandleFunc("/api/v1/csrf-token", s.recoverMiddleware(s.auth(s.handleCSRFToken)))
	mux.HandleFunc("/api/v1/stats", s.recoverMiddleware(s.auth(s.handleStats)))
	mux.HandleFunc("/api/v1/devices", s.recoverMiddleware(s.auth(s.handleDevices)))
	mux.HandleFunc("/api/v1/history", s.recoverMiddleware(s.auth(s.handleHistory)))

	// SECURITY FIX LOW-1: Protect state-changing endpoints with CSRF
	// SECURITY FIX #156: Apply write rate limiting to state-changing endpoints
	s.registerWriteProtectedRoutes(mux)
	s.registerReadOnlyRoutes(mux)
	s.registerWalkRoutes(mux)
	s.registerPcapRoutes(mux)
	s.registerSSERoutes(mux)

	// SECURITY FIX #172: Metrics endpoint requires authentication
	mux.HandleFunc("/metrics", s.recoverMiddleware(s.auth(s.handleMetrics)))
	mux.HandleFunc("/", s.recoverMiddleware(s.auth(s.serveSPA())))
}

// registerWriteProtectedRoutes registers routes that require write protection (CSRF + rate limiting).
func (s *Server) registerWriteProtectedRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"/api/v1/config",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleConfig)))),
	)
	mux.HandleFunc(
		"/api/v1/config/devices",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDevicesV2)))),
	)
	mux.HandleFunc(
		"/api/v1/config/devices/",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDevicesV2)))),
	)
	mux.HandleFunc(
		"/api/v1/config/merge",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleConfigMerge)))),
	)
	mux.HandleFunc(
		"/api/v1/config/import",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleConfigImport)))),
	)
	mux.HandleFunc(
		"/api/v1/replay",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleReplay)))),
	)
	mux.HandleFunc(
		"/api/v1/alerts",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleAlerts)))),
	)
	mux.HandleFunc(
		"/api/v1/capture/filter",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleCaptureFilter)))),
	)
	// Standalone packet capture (no simulation required). POST=start,
	// DELETE=stop, GET=status. Distinct from /capture/filter, which
	// drives the BPF filter on the sim-managed capture engine.
	mux.HandleFunc(
		"/api/v1/capture",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleStandaloneCapture)))),
	)
}

// registerReadOnlyRoutes registers routes that only require authentication.
func (s *Server) registerReadOnlyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/config/schema", s.recoverMiddleware(s.auth(s.handleConfigSchema)))
	// SECURITY FIX #171: Apply file-specific rate limiting
	mux.HandleFunc("/api/v1/files", s.recoverMiddleware(s.auth(s.fileRateLimit(s.handleFiles))))

	// Templates API
	mux.HandleFunc("/api/v1/templates", s.recoverMiddleware(s.auth(s.handleTemplates)))
	mux.HandleFunc("/api/v1/templates/", s.recoverMiddleware(s.auth(s.handleTemplateByName)))

	// User Configs API - GET is read-only, POST/DELETE handled separately
	mux.HandleFunc("/api/v1/configs", s.recoverMiddleware(s.auth(s.handleUserConfigs)))
	mux.HandleFunc("/api/v1/configs/", s.recoverMiddleware(s.auth(s.handleUserConfigByName)))

	// Unified Library API (#548). Networks landed in PR 1 with full
	// CRUD; walks/pcaps land in PR 3 as read-only browser endpoints —
	// uploads + deletes for binary content are a separate follow-up
	// once the picker integrations have validated the read shape.
	mux.HandleFunc("/api/v1/library/networks",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleLibraryNetworks)))))
	mux.HandleFunc("/api/v1/library/networks/",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleLibraryNetworkByName)))))
	mux.HandleFunc("/api/v1/library/walks",
		s.recoverMiddleware(s.auth(s.handleLibraryWalks)))
	mux.HandleFunc("/api/v1/library/pcaps",
		s.recoverMiddleware(s.auth(s.handleLibraryPcaps)))

	// Per-device baseline SNMP walk synthesis (#546 part 2). POST-only.
	// Distinct from /api/v1/config/devices/ (the editor's CRUD); this
	// path lives under /api/v1/devices/{hostname}/... and dispatches
	// to a per-action handler. CSRF + rate limiting applied since the
	// handler mutates both the library and the running config YAML.
	mux.HandleFunc("/api/v1/devices/",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.dispatchDeviceSubpath)))))

	// Vendor + model catalog for the Generate-walk UI picker (#570).
	// Read-only; the response is the same list returned by
	// synth.AllModels() so the dropdown can group/sort client-side.
	mux.HandleFunc("/api/v1/synthesize-walk/models",
		s.recoverMiddleware(s.auth(s.handleSynthesizeWalkModels)))

	// Per-type device editor schema (#546 part 1). Read-only — the
	// schema is a static table the daemon serves so the UI can hide
	// sections that don't apply (e.g. switch should not see DNS).
	mux.HandleFunc("/api/v1/device-schemas",
		s.recoverMiddleware(s.auth(s.handleDeviceEditorSchema)))
	mux.HandleFunc("/api/v1/device-schemas/",
		s.recoverMiddleware(s.auth(s.handleDeviceEditorSchema)))
	mux.HandleFunc("/api/v1/topology", s.recoverMiddleware(s.auth(s.handleTopology)))
	mux.HandleFunc("/api/v1/topology/export", s.recoverMiddleware(s.auth(s.handleTopologyExport)))
	mux.HandleFunc("/api/v1/errors", s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleErrors)))))
	mux.HandleFunc("/api/v1/interfaces", s.recoverMiddleware(s.auth(s.handleInterfaces)))
	mux.HandleFunc("/api/v1/runtime", s.recoverMiddleware(s.auth(s.handleRuntime)))
	mux.HandleFunc(
		"/api/v1/simulation",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleSimulation)))),
	)
	mux.HandleFunc("/api/v1/version", s.recoverMiddleware(s.auth(s.handleVersion)))
	mux.HandleFunc("/api/v1/neighbors", s.recoverMiddleware(s.auth(s.handleNeighbors)))
}

// registerWalkRoutes registers SNMP walk file validation endpoints.
func (s *Server) registerWalkRoutes(mux *http.ServeMux) {
	// SECURITY FIX #156: Apply walk-specific rate limiting
	mux.HandleFunc(
		"/api/v1/walk/validate",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.csrfProtect(s.handleWalkValidation)))),
	)
	mux.HandleFunc(
		"/api/v1/walk/fix",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.csrfProtect(s.handleWalkValidation)))),
	)
	mux.HandleFunc(
		"/api/v1/walk/list",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.handleWalkList))),
	)
	mux.HandleFunc(
		"/api/v1/walk/validate-all",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.csrfProtect(s.handleWalkBatchValidate)))),
	)
}

// registerPcapRoutes registers PCAP analysis endpoints.
func (s *Server) registerPcapRoutes(mux *http.ServeMux) {
	// SECURITY FIX #156: Apply upload-specific rate limiting for uploads
	mux.HandleFunc(
		"/api/v1/pcap/upload",
		s.recoverMiddleware(s.auth(s.uploadRateLimit(s.csrfProtect(
			s.requireFeature("pcap_ingest", s.handlePcapUpload))))),
	)
	mux.HandleFunc("/api/v1/pcap/",
		s.recoverMiddleware(s.auth(
			s.requireFeature("pcap_ingest", s.handlePcapAnalysis))))
}

// registerSSERoutes registers Server-Sent Events endpoints for real-time streaming.
func (s *Server) registerSSERoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/stream/packets", s.recoverMiddleware(s.auth(s.handleSSEPackets)))
	mux.HandleFunc("/api/v1/stream/logs", s.recoverMiddleware(s.auth(s.handleSSELogs)))
	mux.HandleFunc("/api/v1/stream/stats", s.recoverMiddleware(s.auth(s.handleSSEStats)))
	mux.HandleFunc("/api/v1/stream/status", s.recoverMiddleware(s.auth(s.handleSSEStatus)))
}

// newSecureHTTPServer creates an HTTP server with security timeouts configured.
func newSecureHTTPServer(addr string, handler http.Handler) *http.Server {
	// SECURITY FIX #99: Add HTTP timeouts to prevent slowloris attacks
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       httpReadTimeout * time.Second,
		WriteTimeout:      httpWriteTimeout * time.Second,
		IdleTimeout:       httpIdleTimeout * time.Second,
		ReadHeaderTimeout: httpReadHeaderTimeout * time.Second,
		MaxHeaderBytes:    MaxRequestBodySize, // 1MB
	}
}

// startBackgroundTasks starts the rate limiter cleanup and SSE hub goroutines.
func (s *Server) startBackgroundTasks() {
	// FEATURE #104: Start periodic cleanup of stale rate limiters
	go func() {
		ticker := time.NewTicker(rateLimiterCleanupMins * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-s.bgStop:
				return
			case <-ticker.C:
				s.rateLimiter.CleanupStale()
			}
		}
	}()

	// Start SSE hub for real-time streaming
	go s.sseHub.Run()
	s.logger.Info("[SSE] Server-Sent Events hub started")

	// Tee slog into the SSE hub so the Protocol Debug Console gets
	// real-time daemon logs. Idempotent: the first call installs a
	// single global wrapper around the then-current slog.Default; later
	// calls (e.g. a second daemon in tests, hot reload) just rotate the
	// active hub via an atomic pointer. That avoids the handler-chain-
	// growth deadlock from the old SetDefault-on-every-Start design.
	// The wrapper filters out "[SSE]"-prefixed records so the hub's
	// own log lines don't recurse.
	InstallSSELogTee(s.sseHub)
}
