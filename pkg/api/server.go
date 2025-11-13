package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/storage"
)

// AlertConfig controls basic threshold-based alerting.
type AlertConfig struct {
	PacketsThreshold uint64
	WebhookURL       string
}

// ServerConfig defines API server options.
type ServerConfig struct {
	Addr        string
	MetricsAddr string
	Token       string
	Stack       *protocols.Stack
	Config      *config.Config
	Storage     *storage.Storage
	Interface   string
	Version     string
	Topology    Topology
	Alert       AlertConfig
}

// Server exposes the REST API, metrics endpoint, and Web UI.
type Server struct {
	cfg           ServerConfig
	httpServer    *http.Server
	metricsServer *http.Server
	alertStop     chan struct{}
	lastAlert     uint64
	mu            sync.Mutex
}

// NewServer returns a configured API server.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Start boots the HTTP listeners.
func (s *Server) Start() error {
	if s.cfg.Stack == nil || s.cfg.Config == nil {
		return fmt.Errorf("api server requires stack and config references")
	}

	if s.cfg.Addr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/stats", s.auth(s.handleStats))
		mux.HandleFunc("/api/v1/devices", s.auth(s.handleDevices))
		mux.HandleFunc("/api/v1/history", s.auth(s.handleHistory))
		mux.HandleFunc("/api/v1/topology", s.auth(s.handleTopology))
		mux.HandleFunc("/api/v1/version", s.auth(s.handleVersion))
		mux.HandleFunc("/metrics", s.handleMetrics)
		mux.HandleFunc("/app.js", s.auth(s.serveStatic("app.js")))
		mux.HandleFunc("/styles.css", s.auth(s.serveStatic("styles.css")))
		mux.HandleFunc("/", s.auth(s.serveStatic("index.html")))

		s.httpServer = &http.Server{
			Addr:    s.cfg.Addr,
			Handler: mux,
		}

		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("api server stopped: %v", err)
			}
		}()
	}

	if s.cfg.MetricsAddr != "" && s.cfg.MetricsAddr != s.cfg.Addr {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", s.handleMetrics)

		s.metricsServer = &http.Server{
			Addr:    s.cfg.MetricsAddr,
			Handler: mux,
		}

		go func() {
			if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("metrics server stopped: %v", err)
			}
		}()
	}

	if s.cfg.Alert.PacketsThreshold > 0 {
		s.alertStop = make(chan struct{})
		go s.alertLoop()
	}

	return nil
}

// Shutdown stops the HTTP listeners.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.alertStop != nil {
		close(s.alertStop)
	}

	if s.metricsServer != nil {
		_ = s.metricsServer.Shutdown(ctx)
	}

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token == "" {
			next(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.URL.Query().Get("token")
		}

		if token != s.cfg.Token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) serveStatic(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(uiFS, path.Join("ui", name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.cfg.Stack.GetStats()
	payload := map[string]interface{}{
		"timestamp":    time.Now().UTC(),
		"interface":    s.cfg.Interface,
		"version":      s.cfg.Version,
		"device_count": len(s.cfg.Config.Devices),
		"stack": map[string]uint64{
			"packets_sent":     stats.PacketsSent,
			"packets_received": stats.PacketsReceived,
			"arp_requests":     stats.ARPRequests,
			"arp_replies":      stats.ARPReplies,
			"icmp_requests":    stats.ICMPRequests,
			"icmp_replies":     stats.ICMPReplies,
			"dns_queries":      stats.DNSQueries,
			"dhcp_requests":    stats.DHCPRequests,
			"snmp_queries":     stats.SNMPQueries,
			"errors":           stats.Errors,
		},
	}
	s.writeJSON(w, payload)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	devices := make([]map[string]interface{}, 0, len(s.cfg.Config.Devices))
	for _, dev := range s.cfg.Config.Devices {
		ips := make([]string, 0, len(dev.IPAddresses))
		for _, ip := range dev.IPAddresses {
			ips = append(ips, ip.String())
		}

		protos := make([]string, 0, 8)
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			protos = append(protos, "SNMP")
		}
		if dev.DHCPConfig != nil {
			protos = append(protos, "DHCP")
		}
		if dev.DNSConfig != nil {
			protos = append(protos, "DNS")
		}
		if dev.HTTPConfig != nil {
			protos = append(protos, "HTTP")
		}
		if dev.FTPConfig != nil {
			protos = append(protos, "FTP")
		}
		if dev.LLDPConfig != nil && dev.LLDPConfig.Enabled {
			protos = append(protos, "LLDP")
		}
		if dev.CDPConfig != nil && dev.CDPConfig.Enabled {
			protos = append(protos, "CDP")
		}

		devices = append(devices, map[string]interface{}{
			"name":      dev.Name,
			"type":      dev.Type,
			"ips":       ips,
			"protocols": protos,
		})
	}
	s.writeJSON(w, devices)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Storage == nil {
		s.writeJSON(w, []storage.RunRecord{})
		return
	}
	history, err := s.cfg.Storage.ListRuns(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, history)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, s.cfg.Topology)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]string{"version": s.cfg.Version})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats := s.cfg.Stack.GetStats()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "niac_packets_sent_total %d\n", stats.PacketsSent)
	fmt.Fprintf(w, "niac_packets_received_total %d\n", stats.PacketsReceived)
	fmt.Fprintf(w, "niac_snmp_queries_total %d\n", stats.SNMPQueries)
	fmt.Fprintf(w, "niac_errors_total %d\n", stats.Errors)
	fmt.Fprintf(w, "niac_devices_total %d\n", len(s.cfg.Config.Devices))
}

func (s *Server) writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func (s *Server) alertLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := s.cfg.Stack.GetStats()
			total := stats.PacketsSent + stats.PacketsReceived
			if total >= s.cfg.Alert.PacketsThreshold {
				s.mu.Lock()
				if total != s.lastAlert {
					s.lastAlert = total
					go s.sendAlert(total)
				}
				s.mu.Unlock()
			}
		case <-s.alertStop:
			return
		}
	}
}

func (s *Server) sendAlert(total uint64) {
	log.Printf("alert: packet threshold exceeded (total=%d)", total)
	if s.cfg.Alert.WebhookURL == "" {
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"type":        "packet_threshold",
		"threshold":   s.cfg.Alert.PacketsThreshold,
		"total":       total,
		"interface":   s.cfg.Interface,
		"triggeredAt": time.Now().UTC(),
	})

	req, err := http.NewRequest(http.MethodPost, s.cfg.Alert.WebhookURL, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("alert webhook error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	if resp, err := client.Do(req); err != nil {
		log.Printf("alert webhook request failed: %v", err)
	} else {
		resp.Body.Close()
	}
}
