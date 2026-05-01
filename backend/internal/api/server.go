package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/obhod/obhoud/internal/service"
	"github.com/obhod/obhoud/internal/uci"
)

type Server struct {
	Manager    *service.Manager
	SocketPath string
	srv        *http.Server
}

func NewServer(mgr *service.Manager) *Server {
	return &Server{
		Manager:    mgr,
		SocketPath: "/var/run/obhoud.sock",
	}
}

type StatusResponse struct {
	Running       bool          `json:"running"`
	UptimeSeconds int           `json:"uptime_seconds"`
	MemoryUsageMb float64       `json:"memory_usage_mb"`
	WanReady      bool          `json:"wan_ready"`
	DnsHealthy    bool          `json:"dns_healthy"`
	ActiveTunnels []TunnelState `json:"active_tunnels"`
}

type TunnelState struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
}

func (s *Server) Start() error {
	// Remove existing socket if it exists
	if err := os.RemoveAll(s.SocketPath); err != nil {
		return err
	}

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return err
	}

	// Set permissions
	if err := os.Chmod(s.SocketPath, 0660); err != nil {
		log.Printf("Warning: failed to chmod socket: %v", err)
	}
	// Note: to change owner to root:luci, we would need chown
	// e.g. os.Chown(s.SocketPath, 0, luci_uid) - needs lookup in real implementation

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/tunnels", s.handleTunnels)
	mux.HandleFunc("/api/rules", s.handleRules)
	mux.HandleFunc("/api/restart", s.handleRestart)

	s.srv = &http.Server{Handler: mux}

	log.Printf("API Server listening on %s", s.SocketPath)
	go func() {
		if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("API Server error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping API Server...")
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tunnels := []TunnelState{}
	for _, ts := range s.Manager.GetTunnelsStatus() {
		status := "connected"
		if ts.Status == "down" {
			status = "disconnected"
		}
		tunnels = append(tunnels, TunnelState{
			ID:        ts.ID,
			Status:    status,
			LatencyMs: ts.LatencyMs,
		})
	}

	resp := StatusResponse{
		Running:       true,
		UptimeSeconds: s.Manager.UptimeSeconds(),
		MemoryUsageMb: 0, // Not implemented yet
		WanReady:      s.Manager.IsWANReady(),
		DnsHealthy:    s.Manager.IsDNSHealthy(),
		ActiveTunnels: tunnels,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var tunnels []uci.Tunnel
	if s.Manager.UciConfig != nil {
		tunnels = s.Manager.UciConfig.Tunnels
	}
	json.NewEncoder(w).Encode(tunnels)
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var rules []uci.RoutingRule
	if s.Manager.UciConfig != nil {
		rules = s.Manager.UciConfig.Routing
	}
	json.NewEncoder(w).Encode(rules)
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.Manager.RestartSingBox(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
