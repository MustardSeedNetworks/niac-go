package api

import (
	"net/http"
	"time"
)

// registerAPIRoutes registers all API endpoints on the provided mux.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
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
	s.registerTemplateRoutes(mux)
	s.registerDebugRoutes(mux)
	s.registerSSERoutes(mux)

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
		"/api/v1/replay",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleReplay)))),
	)
	mux.HandleFunc(
		"/api/v1/alerts",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleAlerts)))),
	)
}

// registerReadOnlyRoutes registers routes that only require authentication.
func (s *Server) registerReadOnlyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/config/schema", s.recoverMiddleware(s.auth(s.handleConfigSchema)))
	mux.HandleFunc("/api/v1/files", s.recoverMiddleware(s.auth(s.handleFiles)))
	mux.HandleFunc("/api/v1/topology", s.recoverMiddleware(s.auth(s.handleTopology)))
	mux.HandleFunc("/api/v1/topology/export", s.recoverMiddleware(s.auth(s.handleTopologyExport)))
	// FIX #291: Split /api/v1/errors - GET is read-only, POST/DELETE need write protection
	mux.HandleFunc("/api/v1/errors", s.recoverMiddleware(s.auth(s.handleErrorsRouter)))
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
		s.recoverMiddleware(s.auth(s.uploadRateLimit(s.csrfProtect(s.handlePcapUpload)))),
	)
	mux.HandleFunc("/api/v1/pcap/", s.recoverMiddleware(s.auth(s.handlePcapAnalysis)))
}

// registerSSERoutes registers Server-Sent Events endpoints for real-time streaming.
func (s *Server) registerSSERoutes(mux *http.ServeMux) {
	// FIX #283: SSE token endpoint for short-lived authentication tokens
	mux.HandleFunc("/api/v1/stream/token", s.recoverMiddleware(s.auth(s.handleSSEToken)))
	mux.HandleFunc("/api/v1/stream/packets", s.recoverMiddleware(s.auth(s.handleSSEPackets)))
	mux.HandleFunc("/api/v1/stream/logs", s.recoverMiddleware(s.auth(s.handleSSELogs)))
	mux.HandleFunc("/api/v1/stream/stats", s.recoverMiddleware(s.auth(s.handleSSEStats)))
	mux.HandleFunc("/api/v1/stream/status", s.recoverMiddleware(s.auth(s.handleSSEStatus)))
}

// registerTemplateRoutes registers template management endpoints.
// FIX #263: Implements backend for template CRUD operations.
func (s *Server) registerTemplateRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"/api/v1/templates",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleTemplates)))),
	)
	mux.HandleFunc(
		"/api/v1/templates/use",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleTemplateUse)))),
	)
	mux.HandleFunc(
		"/api/v1/templates/",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleTemplateByName)))),
	)
}

// registerDebugRoutes registers protocol debug level endpoints.
// FIX #264: Implements backend for debug level management.
func (s *Server) registerDebugRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"/api/v1/debug/levels",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDebugLevels)))),
	)
	mux.HandleFunc(
		"/api/v1/debug/levels/reset",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDebugLevelsReset)))),
	)
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
		MaxHeaderBytes:    MaxHeaderBytesSize, // 64KB - SECURITY FIX #282
	}
}
