package config

import (
	"bytes"
	"testing"
)

func TestDecodeBase64Tolerant(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:  "Standard Base64",
			input: "SGVsbG8gV29ybGQ=",
			want:  []byte("Hello World"),
		},
		{
			name:  "URL-safe Base64 with padding removed",
			input: "SGVsbG8gV29ybGQ",
			want:  []byte("Hello World"),
		},
		{
			name:  "URL-safe with - and _",
			input: "eyJhZGQiOiIxMjcuMC4wLjEiLCJwb3J0Ijo4MDgwLCJpZCI6ImFhYWEifQ", // {"add":"127.0.0.1","port":8080,"id":"aaaa"}
			want:  []byte(`{"add":"127.0.0.1","port":8080,"id":"aaaa"}`),
		},
		{
			name:  "Base64 with newlines and spaces",
			input: "SGVs bG8g\nV29ybGQ=",
			want:  []byte("Hello World"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeBase64Tolerant(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeBase64Tolerant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !bytes.Equal(got, tt.want) {
				t.Errorf("DecodeBase64Tolerant() = %s, want %s", string(got), string(tt.want))
			}
		})
	}
}

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		name     string
		proxyStr string
		tag      string
		wantType string
		wantPort int
		wantErr  bool
	}{
		{
			name:     "VLESS URL",
			proxyStr: "vless://ae374246-862d-45f8-b395-926f63459ad2@example.com:443?security=tls&sni=example.com&type=ws&path=/vless-path#VlessNode",
			tag:      "vless-node",
			wantType: "vless",
			wantPort: 443,
		},
		{
			name:     "Shadowsocks URL - Standard userinfo",
			proxyStr: "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@example.com:8388#ShadowsocksNode", // aes-256-gcm:password
			tag:      "ss-node",
			wantType: "shadowsocks",
			wantPort: 8388,
		},
		{
			name:     "Trojan URL",
			proxyStr: "trojan://password@example.com:443?security=tls&sni=example.com#TrojanNode",
			tag:      "trojan-node",
			wantType: "trojan",
			wantPort: 443,
		},
		{
			name:     "VMess URL (V2RayN JSON Format)",
			proxyStr: "vmess://eyJhZGQiOiJleGFtcGxlLmNvbSIsImFpZCI6MCwiYWxwbiI6IiIsImZwIjoiIiwiaG9zdCI6IiIsImlkIjoiYWUzNzQyNDYtODYyZC00NWY4LWIzOTUtOTI2ZjYzNDU5YWQyIiwiaW5zZWMiOjAsIm5ldCI6IndzIiwicGF0aCI6Ii92bWVzcy1wYXRoIiwicG9ydCI6NDQzLCJwc2kiOiJWbWVzc05vZGUiLCJzY3kiOiJhdXRvIiwic25pIjoiZXhhbXBsZS5jb20iLCJ0bHMiOiJ0bHMiLCJ0eXBlIjoiIiwidmVyIjoiMiJ9",
			tag:      "vmess-node",
			wantType: "vmess",
			wantPort: 443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProxyURL(tt.proxyStr, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseProxyURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Fatalf("parseProxyURL() returned nil outbound")
				}
				if got.Type != tt.wantType {
					t.Errorf("parseProxyURL() got type = %s, want = %s", got.Type, tt.wantType)
				}
				if got.ServerPort != tt.wantPort {
					t.Errorf("parseProxyURL() got port = %d, want = %d", got.ServerPort, tt.wantPort)
				}
				if got.Tag != tt.tag {
					t.Errorf("parseProxyURL() got tag = %s, want = %s", got.Tag, tt.tag)
				}
			}
		})
	}
}

func TestParseSettingsUCI_ListAppend(t *testing.T) {
	// Tests list append behavior which fixes multi-interface parsing bugs (v1.1.15)
	s := &SettingsUCI{}

	// Mimic parser loop calling parseSettings for two different values
	parseSettings(s, []string{"obhod", "settings", "source_network_interfaces"}, "lan")
	parseSettings(s, []string{"obhod", "settings", "source_network_interfaces"}, "guest")

	if len(s.SourceNetworkInterfaces) != 2 {
		t.Fatalf("Expected 2 source network interfaces, got %d", len(s.SourceNetworkInterfaces))
	}
	if s.SourceNetworkInterfaces[0] != "lan" || s.SourceNetworkInterfaces[1] != "guest" {
		t.Errorf("Unexpected interfaces list content: %v", s.SourceNetworkInterfaces)
	}

	// Mimic space-separated list
	s2 := &SettingsUCI{}
	parseSettings(s2, []string{"obhod", "settings", "source_network_interfaces"}, "lan guest")
	if len(s2.SourceNetworkInterfaces) != 2 {
		t.Fatalf("Expected 2 source network interfaces from space-separated string, got %d", len(s2.SourceNetworkInterfaces))
	}
	if s2.SourceNetworkInterfaces[0] != "lan" || s2.SourceNetworkInterfaces[1] != "guest" {
		t.Errorf("Unexpected interfaces list content: %v", s2.SourceNetworkInterfaces)
	}
}

func TestNewFeatures(t *testing.T) {
	// 1. VLESS packetEncoding test
	got, err := parseProxyURL("vless://ae374246-862d-45f8-b395-926f63459ad2@example.com:443?packetEncoding=xudp", "vless-pe")
	if err != nil {
		t.Fatalf("Failed to parse VLESS with packetEncoding: %v", err)
	}
	if got.PacketEncoding != "xudp" {
		t.Errorf("Expected PacketEncoding = xudp, got %s", got.PacketEncoding)
	}
	if got.Security != "" {
		t.Errorf("Expected Security to be empty for VLESS, got %s", got.Security)
	}

	// 2. VMess plain URL fallback test
	gotVmess, err := parseProxyURL("vmess://ae374246-862d-45f8-b395-926f63459ad2@example.com:8080?security=aes-128-gcm&packetEncoding=xudp", "vmess-fallback")
	if err != nil {
		t.Fatalf("Failed to parse fallback VMess URL: %v", err)
	}
	if gotVmess.Type != "vmess" || gotVmess.Server != "example.com" || gotVmess.ServerPort != 8080 || gotVmess.UUID != "ae374246-862d-45f8-b395-926f63459ad2" {
		t.Errorf("VMess fallback parse mismatch: %+v", gotVmess)
	}
	if gotVmess.Security != "aes-128-gcm" {
		t.Errorf("Expected VMess security = aes-128-gcm, got %s", gotVmess.Security)
	}
	if gotVmess.PacketEncoding != "xudp" {
		t.Errorf("Expected VMess PacketEncoding = xudp, got %s", gotVmess.PacketEncoding)
	}

	// 3. Socks URL parsing test
	gotSocks, err := parseProxyURL("socks5://user:pass@example.com:1080", "socks-test")
	if err != nil {
		t.Fatalf("Failed to parse Socks: %v", err)
	}
	if gotSocks.Type != "socks" || gotSocks.Server != "example.com" || gotSocks.ServerPort != 1080 || gotSocks.Username != "user" || gotSocks.Password != "pass" || gotSocks.Version != "5" {
		t.Errorf("Socks parse mismatch: %+v", gotSocks)
	}

	// 4. setupDNS detour / domain_resolver / rewrite_ttl test
	uci := &UCIConfig{
		Settings: SettingsUCI{
			DNSServer:          "dns.google/dns-query", // domain name, requires domain_resolver
			BootstrapDNSServer: "77.88.8.8",
			DNSType:            "doh",
			DNSRewriteTTL:      120,
		},
		Sections: map[string]SectionUCI{
			"proxy1": {
				Name:           "proxy1",
				Enabled:        true,
				ConnectionType: "proxy",
			},
		},
	}
	cfg := &SingBoxConfig{
		DNS: &DNSConfig{
			Servers: []DNSServerConfig{},
			Rules:   []DNSRuleConfig{},
		},
		Route: &RouteConfig{},
	}
	setupDNS(cfg, uci)
	
	// Verify dns-proxy server configuration
	var dnsProxy *DNSServerConfig
	for i := range cfg.DNS.Servers {
		if cfg.DNS.Servers[i].Tag == "dns-proxy" {
			dnsProxy = &cfg.DNS.Servers[i]
		}
	}
	if dnsProxy == nil {
		t.Fatalf("dns-proxy server not found in DNS configuration")
	}
	if dnsProxy.Type != "https" {
		t.Errorf("Expected dns-proxy Type = https, got %s", dnsProxy.Type)
	}
	if dnsProxy.Detour != "proxy1-out" {
		t.Errorf("Expected dns-proxy Detour = proxy1-out, got %s", dnsProxy.Detour)
	}
	if dnsProxy.DomainResolver != "dns-direct" {
		t.Errorf("Expected dns-proxy DomainResolver = dns-direct, got %s", dnsProxy.DomainResolver)
	}

	// Verify rewrite_ttl on fakeip DNS rule
	var fakeipRule *DNSRuleConfig
	for i := range cfg.DNS.Rules {
		if cfg.DNS.Rules[i].Server == "fakeip-server" {
			fakeipRule = &cfg.DNS.Rules[i]
		}
	}
	if fakeipRule == nil {
		t.Fatalf("fakeip-server rule not found in DNS rules")
	}
	if fakeipRule.RewriteTTL != 120 {
		t.Errorf("Expected fakeip rule RewriteTTL = 120, got %d", fakeipRule.RewriteTTL)
	}
}
