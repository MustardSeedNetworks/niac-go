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
// registered directly. The SPA shell is also public so it can collect a bearer
// token in browser memory before calling the protected API.
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
		{
			path:    "/api/v1/license",
			handler: s.handleLicenseStatus,
			methods: []string{http.MethodGet},
		},
	})

	s.registerSessionRoutes(mux)
	s.registerWriteProtectedRoutes(mux)
	s.registerReadOnlyRoutes(mux)
	s.registerLibraryRoutes(mux)
	s.registerScenarioRoutes(mux)
	s.registerWalkRoutes(mux)
	s.registerPcapRoutes(mux)
	s.registerSSERoutes(mux)

	// Metrics require auth (#172).
	s.registerAll(mux, []apiRoute{
		{path: "/metrics", handler: s.handleMetrics, methods: []string{http.MethodGet}},
		{
			path:    "/api/",
			handler: s.handleAPINotFound,
			methods: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
		},
	})

	// Static assets contain no privileged data. Serving the shell without auth
	// lets the browser prompt for a token; all data still comes from protected
	// /api routes and the token never enters a URL.
	mux.HandleFunc("/", s.recoverMiddleware(
		withSecurityHeaders(
			s.methodGate([]string{http.MethodGet, http.MethodHead}, s.serveSPA()),
		),
	))
}

func (s *Server) registerScenarioRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{
			path:    "/api/v1/scenario/packs",
			handler: s.handleScenarioPacks,
			methods: []string{http.MethodGet},
			feature: "config_templates",
		},
		{
			path:    "/api/v1/scenario/profiles",
			handler: s.handleScenarioProfiles,
			methods: []string{http.MethodGet},
			feature: "config_templates",
		},
		{
			path:    "/api/v1/scenario/profiles/captured",
			handler: s.handleCapturedProfileCreate,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
			feature: "config_templates",
		},
		{
			path:    "/api/v1/scenario/generate",
			handler: s.handleScenarioGenerate,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
			feature: "config_templates",
		},
	})
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "not_found", "API endpoint not found", nil)
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
		// MaxPCAPUploadBodySize, which accounts for base64 + JSON envelope
		// overhead on top of the MaxPCAPUploadSize raw cap), so the registry
		// cap must match that, not 1MB.
		{
			path:         "/api/v1/replay",
			handler:      s.handleReplay,
			methods:      []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			maxBodyBytes: MaxPCAPUploadBodySize,
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
		// Global debug verbosity (GET current + default, PUT to set). The stack
		// reads the global level live, so PUT takes effect with no restart.
		{
			path:    "/api/v1/debug/level",
			handler: s.handleDebugLevel,
			methods: []string{http.MethodGet, http.MethodPut},
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

// registerLibraryRoutes registers the content library surface (#548): networks
// full CRUD, walks/pcaps read-only listing, and the walk mutation actions
// (revert, sanitize) that carry write rate limit + CSRF like the networks
// POST above. Split out of registerReadOnlyRoutes to keep both under the
// funlen cap as the library surface grows (#950).
func (s *Server) registerLibraryRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{
			path:         "/api/v1/library/drafts",
			handler:      s.handleLibraryDrafts,
			methods:      []string{http.MethodGet, http.MethodPost},
			maxBodyBytes: MaxScenarioRequestBodySize,
			rl:           rlWrite,
			csrf:         true,
		},
		{
			path:         "/api/v1/library/drafts/",
			handler:      s.handleLibraryDraftByName,
			methods:      []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete},
			maxBodyBytes: MaxScenarioRequestBodySize,
			rl:           rlWrite,
			csrf:         true,
		},
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
		{
			path:    "/api/v1/library/walks",
			handler: s.handleLibraryWalks,
			methods: []string{http.MethodGet},
		},
		// Revert mutates the walk on disk (restores + removes the .orig
		// sidecar), so — like the networks POST above — it carries write
		// rate limit + CSRF rather than being GET-only like its sibling.
		{
			path:    "/api/v1/library/walks/revert",
			handler: s.handleLibraryWalkRevert,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		// Sanitize mutates the walk on disk (preserves the original, then
		// overwrites with a scrubbed copy — see library.PreserveOriginal),
		// so it carries the same write rate limit + CSRF as revert (#950).
		{
			path:    "/api/v1/library/walks/sanitize",
			handler: s.handleLibraryWalkSanitize,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/library/walks/sanitize-batch",
			handler: s.handleLibraryWalkSanitizeBatch,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/library/pcaps",
			handler: s.handleLibraryPcaps,
			methods: []string{http.MethodGet},
		},
		// Install accepts a gzip-tar content bundle (base64 in the JSON body,
		// like /api/v1/pcap/upload) and extracts it over the whole library —
		// networks/walks/pcaps at once — so it carries the same admin-class
		// policy as /api/v1/config/import: write rate limit, CSRF, AND an
		// admin-scoped token (#897 L3b), plus the larger body cap the base64
		// expansion needs (see MaxLibraryInstallBodySize).
		{
			path:         "/api/v1/library/install",
			handler:      s.handleLibraryInstall,
			methods:      []string{http.MethodPost},
			maxBodyBytes: MaxLibraryInstallBodySize,
			rl:           rlWrite,
			csrf:         true,
			admin:        true,
		},
	})
}

// registerReadOnlyRoutes registers reads plus the mutating CRUD endpoints that
// historically shared this group; the mutating ones carry write rate limit +
// CSRF (#740). csrfProtect internally skips GET, so reads pass through.
func (s *Server) registerReadOnlyRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{
			path:    "/api/v1/config/schema",
			handler: s.handleConfigSchema,
			methods: []string{http.MethodGet},
		},
		{
			path:    "/api/v1/files",
			handler: s.handleFiles,
			methods: []string{http.MethodGet},
			rl:      rlFile,
		},
		// Templates: POST (upload), DELETE, and "use" mutate — write + CSRF.
		{
			path:    "/api/v1/templates",
			handler: s.handleTemplates,
			methods: []string{http.MethodGet, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		{
			path:    "/api/v1/templates/use",
			handler: s.handleTemplateUse,
			methods: []string{http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
			feature: "config_templates",
		},
		{
			path:    "/api/v1/templates/",
			handler: s.handleTemplateByName,
			methods: []string{http.MethodGet, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
		// Per-device actions. synthesize-walk (#546 p2) mutates the library +
		// running config YAML, so this path carries write rate limit + CSRF;
		// csrf.Protect skips safe GETs, so the read-only interfaces action
		// (#897 p5f) added alongside it isn't CSRF-gated in practice.
		{
			path:    "/api/v1/devices/",
			handler: s.dispatchDeviceSubpath,
			methods: []string{http.MethodGet, http.MethodPost},
			rl:      rlWrite,
			csrf:    true,
		},
		// Read-only catalogs / schemas.
		{
			path:    "/api/v1/synthesize-walk/models",
			handler: s.handleSynthesizeWalkModels,
			methods: []string{http.MethodGet},
		},
		{
			path:    "/api/v1/device-schemas",
			handler: s.handleDeviceEditorSchema,
			methods: []string{http.MethodGet},
		},
		{
			path:    "/api/v1/device-schemas/",
			handler: s.handleDeviceEditorSchema,
			methods: []string{http.MethodGet},
		},
	})
	s.registerTopologyReadOnlyRoutes(mux)
}

func (s *Server) registerTopologyReadOnlyRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{path: "/api/v1/topology", handler: s.handleTopology, methods: []string{http.MethodGet}},
		{
			path:    "/api/v1/topology/export",
			handler: s.handleTopologyExport,
			methods: []string{http.MethodGet},
		},
		{path: "/api/v1/segments", handler: s.handleSegments, methods: []string{http.MethodGet}},
		{
			path:    "/api/v1/errors",
			handler: s.handleErrors,
			methods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
			},
			rl:               rlWrite,
			csrf:             true,
			feature:          "error_injection",
			featureWriteOnly: true,
		},
		{
			path:    "/api/v1/interfaces",
			handler: s.handleInterfaces,
			methods: []string{http.MethodGet},
		},
		{path: "/api/v1/runtime", handler: s.handleRuntime, methods: []string{http.MethodGet}},
		{path: "/api/v1/behaviors", handler: s.handleBehaviors, methods: []string{http.MethodGet}},
		{
			path:         "/api/v1/simulation/preflight",
			handler:      s.handleSimulationPreflight,
			methods:      []string{http.MethodPost},
			maxBodyBytes: MaxScenarioRequestBodySize,
			rl:           rlWrite,
			csrf:         true,
		},
		{
			path:         "/api/v1/simulation",
			handler:      s.handleSimulation,
			methods:      []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			maxBodyBytes: MaxScenarioRequestBodySize,
			rl:           rlWrite,
			csrf:         true,
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
			path:         "/api/v1/walk/import",
			handler:      s.handleWalkImport,
			methods:      []string{http.MethodPost},
			rl:           rlWalk,
			csrf:         true,
			feature:      "config_templates",
			maxBodyBytes: MaxWalkImportBodySize,
		},
		{
			path:    "/api/v1/walk/capture-profile",
			handler: s.handleWalkCaptureProfile,
			methods: []string{http.MethodPost},
			rl:      rlWalk,
			csrf:    true,
			feature: "config_templates",
		},
		{
			path:    "/api/v1/walk/validate",
			handler: s.handleWalkValidation,
			methods: []string{http.MethodPost},
			rl:      rlWalk,
			csrf:    true,
		},
		{
			path:    "/api/v1/walk/analyze",
			handler: s.handleWalkAnalyze,
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
		{
			path:    "/api/v1/walk/list",
			handler: s.handleWalkList,
			methods: []string{http.MethodGet},
			rl:      rlWalk,
		},
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
		// Upload decodes a base64 PCAP payload up to MaxPCAPUploadSize (100MB
		// raw) via decodeJSONStrict, but the base64-encoded JSON body is
		// larger than that (~137MB); the registry cap must match
		// MaxPCAPUploadBodySize so it never truncates a valid capture before
		// the handler reads it.
		{
			path:         "/api/v1/pcap/upload",
			handler:      s.handlePcapUpload,
			methods:      []string{http.MethodPost},
			maxBodyBytes: MaxPCAPUploadBodySize,
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
		{
			path:    "/api/v1/stream/packets",
			handler: s.handleSSEPackets,
			methods: []string{http.MethodGet},
		},
		{path: "/api/v1/stream/logs", handler: s.handleSSELogs, methods: []string{http.MethodGet}},
		{
			path:    "/api/v1/stream/stats",
			handler: s.handleSSEStats,
			methods: []string{http.MethodGet},
		},
		{
			path:    "/api/v1/stream/status",
			handler: s.handleSSEStatus,
			methods: []string{http.MethodGet},
		},
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
	go s.startStatsPublisher(statsStreamInterval)
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
