package config

import (
	"encoding/json"
	"github.com/obhod/obhoud/internal/uci"
	"testing"
)

func TestGenerateSingBoxConfig(t *testing.T) {
	cfg := &uci.UciConfig{
		Settings: uci.GlobalSettings{
			LogLevel:   "debug",
			DnsPort:    15353,
			TproxyPort: 11080,
		},
		Tunnels: []uci.Tunnel{
			{
				ID:     "main",
				Type:   "vless",
				Server: "proxy.example.com",
				Port:   443,
				UUID:   "uuid-1234",
			},
		},
		Routing: []uci.RoutingRule{
			{
				ID:      "rule1",
				Target:  "tunnel_main",
				Domains: []string{"example.com", "test.org"},
			},
		},
	}

	data, err := GenerateSingBoxConfig(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Validate it parses back to struct
	var sb SingBoxConfig
	if err := json.Unmarshal(data, &sb); err != nil {
		t.Fatalf("failed to unmarshal generated JSON: %v", err)
	}

	if sb.Log.Level != "debug" {
		t.Errorf("expected log level 'debug', got %s", sb.Log.Level)
	}

	if len(sb.Inbounds) != 2 {
		t.Fatalf("expected 2 inbounds, got %d", len(sb.Inbounds))
	}

	hasTproxy := false
	hasDns := false
	for _, in := range sb.Inbounds {
		if in.Type == "tproxy" && in.ListenPort == 11080 {
			hasTproxy = true
		}
		if in.Type == "direct" && in.Tag == "dns-in" && in.ListenPort == 15353 {
			hasDns = true
		}
	}
	if !hasTproxy {
		t.Errorf("missing tproxy inbound with port 11080")
	}
	if !hasDns {
		t.Errorf("missing dns-in inbound with port 15353")
	}

	if sb.DNS.FakeIP == nil || !sb.DNS.FakeIP.Enabled {
		t.Errorf("expected FakeIP to be enabled")
	}

	foundTunnel := false
	for _, ob := range sb.Outbounds {
		if ob.Tag == "tunnel_main" {
			foundTunnel = true
			if ob.Type != "vless" {
				t.Errorf("expected type vless, got %s", ob.Type)
			}
		}
	}
	if !foundTunnel {
		t.Errorf("missing tunnel_main outbound")
	}

	foundRule := false
	for _, r := range sb.Route.Rules {
		if r.Outbound == "tunnel_main" {
			foundRule = true
			if len(r.DomainSuffix) != 2 {
				t.Errorf("expected 2 domains, got %d", len(r.DomainSuffix))
			}
		}
	}
	if !foundRule {
		t.Errorf("missing routing rule for tunnel_main")
	}
}
