package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/obhod/obhoud/internal/service"
	"github.com/obhod/obhoud/internal/uci"
)

func TestHandleStatus(t *testing.T) {
	cfg := &uci.UciConfig{
		Tunnels: []uci.Tunnel{
			{ID: "main"},
		},
	}
	mgr := service.NewManager(cfg, []byte{})
	srv := NewServer(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	srv.handleStatus(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}

	var resp StatusResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Running {
		t.Errorf("expected running to be true")
	}
	if len(resp.ActiveTunnels) != 1 {
		t.Fatalf("expected 1 active tunnel, got %d", len(resp.ActiveTunnels))
	}
	if resp.ActiveTunnels[0].ID != "main" {
		t.Errorf("expected tunnel ID 'main', got %s", resp.ActiveTunnels[0].ID)
	}
}

func TestServerStartStop(t *testing.T) {
	mgr := service.NewManager(&uci.UciConfig{}, []byte{})
	srv := NewServer(mgr)
	
	// Use a test specific socket path
	srv.SocketPath = "/tmp/obhoud_test.sock"

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}
}
