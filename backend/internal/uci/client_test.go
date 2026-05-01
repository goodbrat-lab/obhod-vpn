package uci

import (
	"testing"
)

func TestParseUciShow(t *testing.T) {
	mockOutput := `
obhod.settings=global
obhod.settings.enabled='1'
obhod.settings.fwmark='100'
obhod.settings.dns_port='5353'
obhod.main=tunnel
obhod.main.type='vless'
obhod.main.server='proxy.example.com'
obhod.main.port='443'
obhod.main.uuid='12345-abcde'
obhod.rule1=routing
obhod.rule1.target='tunnel_main'
obhod.rule1.domains='list1' 'list2'
`

	cfg, err := parseUciShow([]byte(mockOutput))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cfg.Settings.Enabled {
		t.Errorf("expected Enabled to be true")
	}
	if cfg.Settings.Fwmark != 100 {
		t.Errorf("expected Fwmark to be 100, got %d", cfg.Settings.Fwmark)
	}
	if cfg.Settings.DnsPort != 5353 {
		t.Errorf("expected DnsPort to be 5353, got %d", cfg.Settings.DnsPort)
	}

	if len(cfg.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(cfg.Tunnels))
	}
	tunnel := cfg.Tunnels[0]
	if tunnel.ID != "main" {
		t.Errorf("expected tunnel ID 'main', got %s", tunnel.ID)
	}
	if tunnel.Server != "proxy.example.com" {
		t.Errorf("expected server 'proxy.example.com', got %s", tunnel.Server)
	}
	if tunnel.Port != 443 {
		t.Errorf("expected port 443, got %d", tunnel.Port)
	}

	if len(cfg.Routing) != 1 {
		t.Fatalf("expected 1 routing rule, got %d", len(cfg.Routing))
	}
	rule := cfg.Routing[0]
	if rule.Target != "tunnel_main" {
		t.Errorf("expected target 'tunnel_main', got %s", rule.Target)
	}
	if len(rule.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(rule.Domains))
	}
	if rule.Domains[0] != "list1" || rule.Domains[1] != "list2" {
		t.Errorf("unexpected domains: %v", rule.Domains)
	}
}
