package api

import (
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

// registerAPIRoutes registers every endpoint on reg.
//
// Every route — the SPA shell and the introspection endpoints included — is
// installed through the capability registry, which composes its policy (rate
// limiting, auth, method gate, CSRF, admin scope, body cap) in one canonical
// order so a route cannot ship without it. foundation's check-route-policy.sh
// enforces this. /__version and /__capabilities are deliberately
// unauthenticated deployment introspection. The SPA shell is also public so it
// can collect a bearer token in browser memory before calling the protected
// API.
func (s *Server) registerAPIRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{Path: "/__version", Handler: s.handleBuildVersion, Methods: []string{http.MethodGet}},
		{Path: "/__capabilities", Handler: reg.ServeManifest, Methods: []string{http.MethodGet}},
	})

	// Top-level authenticated reads + the CSRF-token endpoint.
	reg.RegisterAll([]route.Route{
		{Path: "/api/v1/csrf-token", Handler: s.handleCSRFToken, Auth: true, Methods: []string{http.MethodGet}},
		{Path: "/api/v1/stats", Handler: s.handleStats, Auth: true, Methods: []string{http.MethodGet}},
		{Path: "/api/v1/devices", Handler: s.handleDevices, Auth: true, Methods: []string{http.MethodGet}},
		{Path: "/api/v1/history", Handler: s.handleHistory, Auth: true, Methods: []string{http.MethodGet}},
	})

	s.registerSessionRoutes(reg)
	s.registerWriteProtectedRoutes(reg)
	s.registerReadOnlyRoutes(reg)
	s.registerLibraryRoutes(reg)
	s.registerScenarioRoutes(reg)
	s.registerWalkRoutes(reg)
	s.registerPcapRoutes(reg)
	s.registerSSERoutes(reg)

	// Metrics require auth (#172).
	reg.RegisterAll([]route.Route{
		{Path: "/metrics", Handler: s.handleMetrics, Auth: true, Methods: []string{http.MethodGet}},
		{
			Path:    "/api/",
			Handler: s.handleAPINotFound,
			Auth:    true,
			// Hidden: a catch-all whose every method would document as an
			// operation a client could call.
			Hidden: true,
			Methods: []string{
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
	// Hidden: the shell is not an API operation, and its every-path pattern
	// would otherwise document as one.
	reg.Register(route.Route{
		Path:    "/",
		Handler: withSecurityHeaders(s.serveSPA()),
		Methods: []string{http.MethodGet, http.MethodHead},
		Hidden:  true,
	})
}

func (s *Server) registerScenarioRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{
			Path:    "/api/v1/scenario/packs",
			Handler: s.handleScenarioPacks,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/api/v1/scenario/profiles",
			Handler: s.handleScenarioProfiles,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/api/v1/scenario/profiles/captured",
			Handler: s.handleCapturedProfileCreate,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/scenario/generate",
			Handler: s.handleScenarioGenerate,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
	})
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "not_found", "API endpoint not found", nil)
}

// registerWriteProtectedRoutes registers state-changing routes (write rate
// limit + CSRF). Whole-topology replacement additionally requires an
// admin-scoped token: /api/v1/config and /config/import. AdminProtect exempts
// safe methods, so /api/v1/config still serves GET to a read-only token.
func (s *Server) registerWriteProtectedRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		// #2173: PUT/PATCH/POST here replace the entire topology and the
		// on-disk config in one shot, which is what ScopeAdmin exists for
		// (tokenstore.ScopeAdmin's doc, #743). GET stays readable to any
		// admitted scope because AdminProtect exempts safe methods. Routine
		// per-device edits are NOT this route — they go to /config/devices/.
		{
			Path:    "/api/v1/config",
			Handler: s.handleConfig,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
			Scope:   scopeAdmin,
		},
		{
			Path:    "/api/v1/config/devices",
			Handler: s.handleDevicesV2,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/config/devices/",
			Handler: s.handleDevicesV2,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/config/merge",
			Handler: s.handleConfigMerge,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		// #743: whole-topology replacement is admin-class (an admin-scoped token
		// in addition to read-write); routine per-device edits / configs CRUD
		// stay at ScopeReadWrite because they are normal operator actions.
		{
			Path:    "/api/v1/config/import",
			Handler: s.handleConfigImport,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
			Scope:   scopeAdmin,
		},
		// Replay accepts inline PCAP payloads (handleReplay POST decodes up to
		// MaxPCAPUploadBodySize, which accounts for base64 + JSON envelope
		// overhead on top of the MaxPCAPUploadSize raw cap), so the registry
		// cap must match that, not 1MB.
		{
			Path:         "/api/v1/replay",
			Handler:      s.handleReplay,
			Auth:         true,
			Methods:      []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			MaxBodyBytes: MaxPCAPUploadBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:    "/api/v1/alerts",
			Handler: s.handleAlerts,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		// Global debug verbosity (GET current + default, PUT to set). The stack
		// reads the global level live, so PUT takes effect with no restart.
		{
			Path:    "/api/v1/debug/level",
			Handler: s.handleDebugLevel,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPut},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/capture/filter",
			Handler: s.handleCaptureFilter,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			Limiter: limitWrite,
			CSRF:    true,
		},
		// Standalone packet capture (POST=start, DELETE=stop, GET=status).
		{
			Path:    "/api/v1/capture",
			Handler: s.handleStandaloneCapture,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
			Limiter: limitWrite,
			CSRF:    true,
		},
	})
}

// registerLibraryRoutes registers the content library surface (#548): networks
// full CRUD, walks/pcaps read-only listing, and the walk mutation actions
// (revert, sanitize) that carry write rate limit + CSRF like the networks
// POST above. Split out of registerReadOnlyRoutes to keep both under the
// funlen cap as the library surface grows (#950).
func (s *Server) registerLibraryRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{
			Path:         "/api/v1/library/drafts",
			Handler:      s.handleLibraryDrafts,
			Auth:         true,
			Methods:      []string{http.MethodGet, http.MethodPost},
			MaxBodyBytes: MaxScenarioRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:         "/api/v1/library/drafts/",
			Handler:      s.handleLibraryDraftByName,
			Auth:         true,
			Methods:      []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete},
			MaxBodyBytes: MaxScenarioRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			// The saved network is a generated scenario, so this route
			// carries the authored-scenario body cap the drafts, preflight
			// and simulation routes carry. On the 1 MiB default, five of the
			// seven shipped packs answered 413 and could not be kept in the
			// library the daemon starts from (#2203).
			Path:         "/api/v1/library/networks",
			Handler:      s.handleLibraryNetworks,
			Auth:         true,
			Methods:      []string{http.MethodGet, http.MethodPost},
			MaxBodyBytes: MaxScenarioRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:    "/api/v1/library/networks/",
			Handler: s.handleLibraryNetworkByName,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodDelete},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/library/walks",
			Handler: s.handleLibraryWalks,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		// Revert mutates the walk on disk (restores + removes the .orig
		// sidecar), so — like the networks POST above — it carries write
		// rate limit + CSRF rather than being GET-only like its sibling.
		{
			Path:    "/api/v1/library/walks/revert",
			Handler: s.handleLibraryWalkRevert,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		// Sanitize mutates the walk on disk (preserves the original, then
		// overwrites with a scrubbed copy — see library.PreserveOriginal),
		// so it carries the same write rate limit + CSRF as revert (#950).
		{
			Path:    "/api/v1/library/walks/sanitize",
			Handler: s.handleLibraryWalkSanitize,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/library/walks/sanitize-batch",
			Handler: s.handleLibraryWalkSanitizeBatch,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/library/pcaps",
			Handler: s.handleLibraryPcaps,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		// Install accepts a gzip-tar content bundle (base64 in the JSON body,
		// like /api/v1/pcap/upload) and extracts it over the whole library —
		// networks/walks/pcaps at once — so it carries the same admin-class
		// policy as /api/v1/config/import: write rate limit, CSRF, AND an
		// admin-scoped token (#897 L3b), plus the larger body cap the base64
		// expansion needs (see MaxLibraryInstallBodySize).
		{
			Path:         "/api/v1/library/install",
			Handler:      s.handleLibraryInstall,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			MaxBodyBytes: MaxLibraryInstallBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
			Scope:        scopeAdmin,
		},
	})
}

// registerReadOnlyRoutes registers reads plus the mutating CRUD endpoints that
// historically shared this group; the mutating ones carry write rate limit +
// CSRF (#740). csrfProtect internally skips GET, so reads pass through.
func (s *Server) registerReadOnlyRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{
			Path:    "/api/v1/config/schema",
			Handler: s.handleConfigSchema,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/api/v1/files",
			Handler: s.handleFiles,
			Auth:    true,
			Methods: []string{http.MethodGet},
			Limiter: limitFile,
		},
		// Templates ship with the product and are read-only; only "use" mutates,
		// so only it carries write rate limit + CSRF.
		{
			Path:    "/api/v1/templates",
			Handler: s.handleTemplates,
			Auth:    true,
			Methods: []string{http.MethodGet},
			Limiter: limitFile,
		},
		{
			Path:    "/api/v1/templates/use",
			Handler: s.handleTemplateUse,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/templates/",
			Handler: s.handleTemplateByName,
			Auth:    true,
			Methods: []string{http.MethodGet},
			Limiter: limitFile,
		},
		// Per-device actions. synthesize-walk (#546 p2) mutates the library +
		// running config YAML, so this path carries write rate limit + CSRF;
		// csrf.Protect skips safe GETs, so the read-only interfaces action
		// (#897 p5f) added alongside it isn't CSRF-gated in practice.
		{
			Path:    "/api/v1/devices/",
			Handler: s.dispatchDeviceSubpath,
			Auth:    true,
			Methods: []string{http.MethodGet, http.MethodPost},
			Limiter: limitWrite,
			CSRF:    true,
		},
		// Read-only catalogs / schemas.
		{
			Path:    "/api/v1/synthesize-walk/models",
			Handler: s.handleSynthesizeWalkModels,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/api/v1/device-schemas",
			Handler: s.handleDeviceEditorSchema,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/api/v1/device-schemas/",
			Handler: s.handleDeviceEditorSchema,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
	})
	s.registerTopologyReadOnlyRoutes(reg)
}

func (s *Server) registerTopologyReadOnlyRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{Path: "/api/v1/topology", Handler: s.handleTopology, Auth: true, Methods: []string{http.MethodGet}},
		{
			Path:    "/api/v1/topology/export",
			Handler: s.handleTopologyExport,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{Path: "/api/v1/segments", Handler: s.handleSegments, Auth: true, Methods: []string{http.MethodGet}},
		{
			Path:         "/api/v1/client-errors",
			Handler:      s.handleClientErrors,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			MaxBodyBytes: MaxRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:    "/api/v1/errors",
			Handler: s.handleErrors,
			Auth:    true,
			Methods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
			},
			Limiter: limitWrite,
			CSRF:    true,
		},
		{
			Path:         "/api/v1/errors/actions",
			Handler:      s.handleDeviceAction,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			MaxBodyBytes: MaxRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:    "/api/v1/interfaces",
			Handler: s.handleInterfaces,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/api/v1/attachment-policies",
			Handler: s.handleAttachmentPolicies,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{Path: "/api/v1/runtime", Handler: s.handleRuntime, Auth: true, Methods: []string{http.MethodGet}},
		{Path: "/api/v1/behaviors", Handler: s.handleBehaviors, Auth: true, Methods: []string{http.MethodGet}},
		{
			Path:         "/api/v1/simulation/attachments",
			Handler:      s.handleSimulationAttachments,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			MaxBodyBytes: MaxScenarioRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:         "/api/v1/simulation/preflight",
			Handler:      s.handleSimulationPreflight,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			MaxBodyBytes: MaxScenarioRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{
			Path:         "/api/v1/simulation",
			Handler:      s.handleSimulation,
			Auth:         true,
			Methods:      []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			MaxBodyBytes: MaxScenarioRequestBodySize,
			Limiter:      limitWrite,
			CSRF:         true,
		},
		{Path: "/api/v1/version", Handler: s.handleVersion, Auth: true, Methods: []string{http.MethodGet}},
		{Path: "/api/v1/neighbors", Handler: s.handleNeighbors, Auth: true, Methods: []string{http.MethodGet}},
		// #762: scope discovery — safe GET, no CSRF / write wrappers needed.
		{Path: "/api/v1/auth/scope", Handler: s.handleAuthScope, Auth: true, Methods: []string{http.MethodGet}},
	})
}

// registerWalkRoutes registers SNMP walk validation endpoints (walk rate limit).
func (s *Server) registerWalkRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{
			Path:         "/api/v1/walk/import",
			Handler:      s.handleWalkImport,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			Limiter:      limitWalk,
			CSRF:         true,
			MaxBodyBytes: MaxWalkImportBodySize,
		},
		{
			Path:    "/api/v1/walk/capture-profile",
			Handler: s.handleWalkCaptureProfile,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWalk,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/walk/validate",
			Handler: s.handleWalkValidation,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWalk,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/walk/analyze",
			Handler: s.handleWalkAnalyze,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWalk,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/walk/fix",
			Handler: s.handleWalkValidation,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWalk,
			CSRF:    true,
		},
		{
			Path:    "/api/v1/walk/list",
			Handler: s.handleWalkList,
			Auth:    true,
			Methods: []string{http.MethodGet},
			Limiter: limitWalk,
		},
		{
			Path:    "/api/v1/walk/validate-all",
			Handler: s.handleWalkBatchValidate,
			Auth:    true,
			Methods: []string{http.MethodPost},
			Limiter: limitWalk,
			CSRF:    true,
		},
	})
}

// registerPcapRoutes registers PCAP analysis endpoints. Upload is
// additionally rate-limited + CSRF-protected.
func (s *Server) registerPcapRoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		// Upload decodes a base64 PCAP payload up to MaxPCAPUploadSize (100MB
		// raw) via decodeJSONStrict, but the base64-encoded JSON body is
		// larger than that (~137MB); the registry cap must match
		// MaxPCAPUploadBodySize so it never truncates a valid capture before
		// the handler reads it.
		{
			Path:         "/api/v1/pcap/upload",
			Handler:      s.handlePcapUpload,
			Auth:         true,
			Methods:      []string{http.MethodPost},
			MaxBodyBytes: MaxPCAPUploadBodySize,
			Limiter:      limitUpload,
			CSRF:         true,
		},
		{
			Path:    "/api/v1/pcap/",
			Handler: s.handlePcapAnalysis,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
	})
}

// registerSSERoutes registers Server-Sent Events streams (auth only).
func (s *Server) registerSSERoutes(reg *route.Registrar) {
	reg.RegisterAll([]route.Route{
		{
			Path:    "/api/v1/stream/packets",
			Handler: s.handleSSEPackets,
			Auth:    true,
			Methods: []string{http.MethodGet},
		},
		{Path: "/api/v1/stream/logs", Handler: s.handleSSELogs, Auth: true, Methods: []string{http.MethodGet}},
		{
			Path:    "/api/v1/stream/status",
			Handler: s.handleSSEStatus,
			Auth:    true,
			Methods: []string{http.MethodGet},
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
