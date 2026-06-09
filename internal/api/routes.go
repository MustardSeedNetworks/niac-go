package api

import (
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

// registerAPIRoutes registers all API endpoints on the provided mux.
//
// Every /api route is installed through the capability registry (register /
// registerAll in route.go), which composes its policy — auth, rate limiting,
// CSRF, admin scope, license feature — in one canonical order so a route cannot
// ship without it. scripts/check-route-policy.sh enforces this. Only the
// unauthenticated introspection endpoints (/__version, /__capabilities) are
// registered directly.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	// Unauthenticated introspection (no auth wrapper).
	mux.HandleFunc("/__version", s.recoverMiddleware(s.handleBuildVersion))
	mux.HandleFunc("/__capabilities", s.recoverMiddleware(s.handleRoutePolicyManifest))

	// Top-level authenticated reads + the CSRF-token endpoint.
	s.registerAll(mux, []apiRoute{
		{path: "/api/v1/csrf-token", handler: s.handleCSRFToken, methods: []string{http.MethodGet}},
		{path: "/api/v1/stats", handler: s.handleStats, methods: []string{http.MethodGet}},
		{path: "/api/v1/devices", handler: s.handleDevices, methods: []string{http.MethodGet}},
		{path: "/api/v1/history", handler: s.handleHistory, methods: []string{http.MethodGet}},
		{path: "/api/v1/license", handler: s.handleLicenseStatus, methods: []string{http.MethodGet}},
	})

	s.registerWriteProtectedRoutes(mux)
	s.registerReadOnlyRoutes(mux)
	s.registerWalkRoutes(mux)
	s.registerPcapRoutes(mux)
	s.registerSSERoutes(mux)

	// Metrics require auth (#172); the SPA is the authenticated catch-all.
	s.registerAll(mux, []apiRoute{
		{path: "/metrics", handler: s.handleMetrics, methods: []string{http.MethodGet}},
		// The SPA serves static assets only; HEAD is honored for asset probes.
		{path: "/", handler: s.serveSPA(), methods: []string{http.MethodGet, http.MethodHead}},
	})
}

// registerWriteProtectedRoutes registers state-changing routes (write rate
// limit + CSRF; /config/import additionally requires an admin-scoped token).
func (s *Server) registerWriteProtectedRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{
			path:    "/api/v1/config",
			handler: s.handleConfig,
			methods: []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/config/devices",
			handler: s.handleDevicesV2,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/config/devices/",
			handler: s.handleDevicesV2,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/config/merge",
			handler: s.handleConfigMerge,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		// #743: whole-topology replacement is admin-class (an admin-scoped token
		// in addition to read-write); routine per-device edits / configs CRUD
		// stay at ScopeReadWrite because they are normal operator actions.
		{
			path:    "/api/v1/config/import",
			handler: s.handleConfigImport,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
			admin:   true,
		},
		// Replay accepts inline PCAP payloads (handleReplay POST decodes up to
		// MaxPCAPUploadSize), so the registry cap must match that, not 1MB.
		{
			path:         "/api/v1/replay",
			handler:      s.handleReplay,
			methods:      []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			maxBodyBytes: MaxPCAPUploadSize,
			rl:           rlWrite,
			csrf:         true,
		},
		{
			path:    "/api/v1/alerts",
			handler: s.handleAlerts,
			methods: []string{http.MethodGet, http.MethodPut, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/capture/filter",
			handler: s.handleCaptureFilter,
			methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		// Standalone packet capture (POST=start, DELETE=stop, GET=status).
		{
			path:    "/api/v1/capture",
			handler: s.handleStandaloneCapture,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
	})
}

// registerReadOnlyRoutes registers reads plus the mutating CRUD endpoints that
// historically shared this group; the mutating ones carry write rate limit +
// CSRF (#740). csrfProtect internally skips GET, so reads pass through.
func (s *Server) registerReadOnlyRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{path: "/api/v1/config/schema", handler: s.handleConfigSchema, methods: []string{http.MethodGet}},
		{path: "/api/v1/files", handler: s.handleFiles, methods: []string{http.MethodGet}, rl: rlFile},
		// Templates: POST (upload), DELETE, and "use" mutate — write + CSRF.
		{
			path:    "/api/v1/templates",
			handler: s.handleTemplates,
			methods: []string{http.MethodGet, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/templates/",
			handler: s.handleTemplateByName,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		// User configs: POST/DELETE mutate — write + CSRF (#740).
		{
			path:    "/api/v1/configs",
			handler: s.handleUserConfigs,
			methods: []string{http.MethodGet, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/configs/",
			handler: s.handleUserConfigByName,
			methods: []string{http.MethodGet, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		// Library (#548): networks full CRUD (write + CSRF); walks/pcaps read-only.
		{
			path:    "/api/v1/library/networks",
			handler: s.handleLibraryNetworks,
			methods: []string{http.MethodGet, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/library/networks/",
			handler: s.handleLibraryNetworkByName,
			methods: []string{http.MethodGet, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		{path: "/api/v1/library/walks", handler: s.handleLibraryWalks, methods: []string{http.MethodGet}},
		{path: "/api/v1/library/pcaps", handler: s.handleLibraryPcaps, methods: []string{http.MethodGet}},
		// Per-device baseline walk synthesis (#546 p2). POST-only; mutates the
		// library + running config YAML, so write + CSRF.
		{
			path:    "/api/v1/devices/",
			handler: s.dispatchDeviceSubpath,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		// Read-only catalogs / schemas.
		{
			path:    "/api/v1/synthesize-walk/models",
			handler: s.handleSynthesizeWalkModels,
			methods: []string{http.MethodGet},
		},
		{path: "/api/v1/device-schemas", handler: s.handleDeviceEditorSchema, methods: []string{http.MethodGet}},
		{path: "/api/v1/device-schemas/", handler: s.handleDeviceEditorSchema, methods: []string{http.MethodGet}},
		{path: "/api/v1/topology", handler: s.handleTopology, methods: []string{http.MethodGet}},
		{path: "/api/v1/topology/export", handler: s.handleTopologyExport, methods: []string{http.MethodGet}},
		{
			path:    "/api/v1/errors",
			handler: s.handleErrors,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		{path: "/api/v1/interfaces", handler: s.handleInterfaces, methods: []string{http.MethodGet}},
		{path: "/api/v1/runtime", handler: s.handleRuntime, methods: []string{http.MethodGet}},
		{
			path:    "/api/v1/simulation",
			handler: s.handleSimulation,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		{path: "/api/v1/version", handler: s.handleVersion, methods: []string{http.MethodGet}},
		{path: "/api/v1/neighbors", handler: s.handleNeighbors, methods: []string{http.MethodGet}},
		// #762: scope discovery — safe GET, no CSRF / write wrappers needed.
		{path: "/api/v1/auth/scope", handler: s.handleAuthScope, methods: []string{http.MethodGet}},
	})
}

// registerWalkRoutes registers SNMP walk validation endpoints (walk rate limit).
func (s *Server) registerWalkRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{
			path:    "/api/v1/walk/validate",
			handler: s.handleWalkValidation,
			methods: []string{http.MethodPost},
			rl:      rlWalk,
			csrf:    true,
		},
		{
			path:    "/api/v1/walk/fix",
			handler: s.handleWalkValidation,
			methods: []string{http.MethodPost},
			rl:      rlWalk,
			csrf:    true,
		},
		{path: "/api/v1/walk/list", handler: s.handleWalkList, methods: []string{http.MethodGet}, rl: rlWalk},
		{
			path:    "/api/v1/walk/validate-all",
			handler: s.handleWalkBatchValidate,
			methods: []string{http.MethodPost},
			rl:      rlWalk,
			csrf:    true,
		},
	})
}

// registerPcapRoutes registers PCAP analysis endpoints (gated by the
// pcap_ingest license feature; upload additionally rate-limited + CSRF).
func (s *Server) registerPcapRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		// Upload decodes a base64 PCAP payload up to MaxPCAPUploadSize (100MB)
		// via decodeJSONStrict; the registry cap must match so it never truncates
		// a valid capture before the handler reads it.
		{
			path:         "/api/v1/pcap/upload",
			handler:      s.handlePcapUpload,
			methods:      []string{http.MethodPost},
			maxBodyBytes: MaxPCAPUploadSize,
			rl:           rlUpload,
			csrf:         true,
			feature:      "pcap_ingest",
		},
		{
			path:    "/api/v1/pcap/",
			handler: s.handlePcapAnalysis,
			methods: []string{http.MethodGet},
			feature: "pcap_ingest",
		},
	})
}

// registerSSERoutes registers Server-Sent Events streams (auth only).
func (s *Server) registerSSERoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{path: "/api/v1/stream/packets", handler: s.handleSSEPackets, methods: []string{http.MethodGet}},
		{path: "/api/v1/stream/logs", handler: s.handleSSELogs, methods: []string{http.MethodGet}},
		{path: "/api/v1/stream/stats", handler: s.handleSSEStats, methods: []string{http.MethodGet}},
		{path: "/api/v1/stream/status", handler: s.handleSSEStatus, methods: []string{http.MethodGet}},
	})
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
	sse.InstallLogTee(s.sseHub)
}
