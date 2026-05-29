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

	// 3b. WS transport host mapping test
	gotWS, err := parseProxyURL("vless://ae374246-862d-45f8-b395-926f63459ad2@example.com:443?security=tls&sni=example.com&type=ws&path=/ws&host=example.com", "vless-ws")
	if err != nil {
		t.Fatalf("Failed to parse VLESS with ws: %v", err)
	}
	if gotWS.Transport == nil {
		t.Fatalf("Expected TransportConfig to be non-nil")
	}
	if gotWS.Transport.Type != "ws" || gotWS.Transport.Path != "/ws" {
		t.Errorf("Unexpected transport type/path: %+v", gotWS.Transport)
	}
	if len(gotWS.Transport.Host) != 0 {
		t.Errorf("Expected Host array to be empty for ws transport, got %v", gotWS.Transport.Host)
	}
	if gotWS.Transport.Headers == nil || gotWS.Transport.Headers["Host"] != "example.com" {
		t.Errorf("Expected Headers to contain Host: example.com, got %v", gotWS.Transport.Headers)
	}

	// 3c. VMess base64 JSON ws transport host mapping test
	// JSON: {"add":"example.com","port":443,"id":"ae374246-862d-45f8-b395-926f63459ad2","net":"ws","path":"/vmess-path","host":"example.com","tls":"tls"}
	// encoded: eyJhZGQiOiJleGFtcGxlLmNvbSIsInBvcnQiOjQ0MywiaWQiOiJhZTM3NDI0Ni04NjJkLTQ1ZjgtYjM5NS05MjZmNjM0NTlhZDIiLCJuZXQiOiJ3cyIsInBhdGgiOiIvdm1lc3MtcGF0aCIsImhvc3QiOiJleGFtcGxlLmNvbSIsInRscyI6InRscyJ9
	gotVMessWS, err := parseProxyURL("vmess://eyJhZGQiOiJleGFtcGxlLmNvbSIsInBvcnQiOjQ0MywiaWQiOiJhZTM3NDI0Ni04NjJkLTQ1ZjgtYjM5NS05MjZmNjM0NTlhZDIiLCJuZXQiOiJ3cyIsInBhdGgiOiIvdm1lc3MtcGF0aCIsImhvc3QiOiJleGFtcGxlLmNvbSIsInRscyI6InRscyJ9", "vmess-ws")
	if err != nil {
		t.Fatalf("Failed to parse VMess base64 JSON with ws: %v", err)
	}
	if gotVMessWS.Transport == nil {
		t.Fatalf("Expected VMess TransportConfig to be non-nil")
	}
	if gotVMessWS.Transport.Type != "ws" || gotVMessWS.Transport.Path != "/vmess-path" {
		t.Errorf("Unexpected VMess transport type/path: %+v", gotVMessWS.Transport)
	}
	if len(gotVMessWS.Transport.Host) != 0 {
		t.Errorf("Expected VMess Host array to be empty for ws transport, got %v", gotVMessWS.Transport.Host)
	}
	if gotVMessWS.Transport.Headers == nil || gotVMessWS.Transport.Headers["Host"] != "example.com" {
		t.Errorf("Expected VMess Headers to contain Host: example.com, got %v", gotVMessWS.Transport.Headers)
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

	// 5. Test mixed inbounds (system and section)
	uciMixed := &UCIConfig{
		Settings: SettingsUCI{
			DownloadListsViaProxy: true,
			ServiceListenAddress:  "192.168.1.1",
		},
		Sections: map[string]SectionUCI{
			"proxy1": {
				Name:               "proxy1",
				Enabled:            true,
				ConnectionType:     "proxy",
				ProxyConfigType:    "url",
				ProxyString:        "vless://ae374246-862d-45f8-b395-926f63459ad2@example.com:443",
				MixedProxyEnabled:  true,
				MixedProxyPort:     8080,
			},
		},
	}
	cfgMixed, err := Generate(uciMixed)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check system service-mixed-in inbound
	var serviceMixed *InboundConfig
	var userMixed *InboundConfig
	for i := range cfgMixed.Inbounds {
		if cfgMixed.Inbounds[i].Tag == "service-mixed-in" {
			serviceMixed = &cfgMixed.Inbounds[i]
		}
		if cfgMixed.Inbounds[i].Tag == "proxy1-mixed" {
			userMixed = &cfgMixed.Inbounds[i]
		}
	}

	if serviceMixed == nil {
		t.Errorf("system mixed inbound (service-mixed-in) was not generated")
	} else {
		if serviceMixed.Type != "mixed" || serviceMixed.Listen != "127.0.0.1" || serviceMixed.ListenPort != 4534 {
			t.Errorf("system mixed inbound configuration mismatch: %+v", serviceMixed)
		}
	}

	if userMixed == nil {
		t.Errorf("user section-level mixed inbound (proxy1-mixed) was not generated")
	} else {
		if userMixed.Type != "mixed" || userMixed.Listen != "192.168.1.1" || userMixed.ListenPort != 8080 {
			t.Errorf("user mixed inbound configuration mismatch: %+v", userMixed)
		}
	}

	// Verify route rules for these inbounds
	var serviceMixedRule *RouteRuleConfig
	var userMixedRule *RouteRuleConfig
	for i := range cfgMixed.Route.Rules {
		r := &cfgMixed.Route.Rules[i]
		if len(r.Inbound) > 0 {
			if r.Inbound[0] == "service-mixed-in" {
				serviceMixedRule = r
			}
			if r.Inbound[0] == "proxy1-mixed" {
				userMixedRule = r
			}
		}
	}

	if serviceMixedRule == nil {
		t.Errorf("route rule for service-mixed-in not found")
	} else if serviceMixedRule.Outbound != "proxy1-out" {
		t.Errorf("expected service-mixed-in route to proxy1-out, got %s", serviceMixedRule.Outbound)
	}

	if userMixedRule == nil {
		t.Errorf("route rule for proxy1-mixed not found")
	} else if userMixedRule.Outbound != "proxy1-out" {
		t.Errorf("expected proxy1-mixed route to proxy1-out, got %s", userMixedRule.Outbound)
	}

	// 6. Test urltest and custom outbound generation
	uciURLTest := &UCIConfig{
		Settings: SettingsUCI{
			Enabled: true,
		},
		Sections: map[string]SectionUCI{
			"urltest_sec": {
				Name:                 "urltest_sec",
				Enabled:              true,
				ConnectionType:       "proxy",
				ProxyConfigType:      "urltest",
				URLTestProxyLinks:    []string{
					"vless://ae374246-862d-45f8-b395-926f63459ad2@example1.com:443",
					"vless://ae374246-862d-45f8-b395-926f63459ad2@example2.com:443",
				},
				URLTestCheckInterval: "5m",
				URLTestTolerance:     30,
				URLTestTestingURL:    "https://www.google.com/generate_204",
			},
			"custom_sec": {
				Name:            "custom_sec",
				Enabled:         true,
				ConnectionType:  "proxy",
				ProxyConfigType: "outbound",
				OutboundJSON:    `{"type":"shadowsocks","server":"example.com","server_port":8388,"method":"aes-256-gcm","password":"pwd"}`,
			},
		},
	}
	cfgURLTest, err := Generate(uciURLTest)
	if err != nil {
		t.Fatalf("Generate failed for urltest/custom: %v", err)
	}

	// Validate urltest_sec outbounds
	var nodes []OutboundConfig
	var urltestGrp *OutboundConfig
	var selectorGrp *OutboundConfig
	var customOut *OutboundConfig

	for i := range cfgURLTest.Outbounds {
		o := &cfgURLTest.Outbounds[i]
		if o.Tag == "urltest_sec-out-1" || o.Tag == "urltest_sec-out-2" {
			nodes = append(nodes, *o)
		} else if o.Tag == "urltest_sec-urltest-out" {
			urltestGrp = o
		} else if o.Tag == "urltest_sec-out" {
			selectorGrp = o
		} else if o.Tag == "custom_sec-out" {
			customOut = o
		}
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 urltest individual nodes, got %d", len(nodes))
	}
	if urltestGrp == nil {
		t.Errorf("urltest group not found")
	} else {
		if urltestGrp.Type != "urltest" || urltestGrp.Interval != "5m" || urltestGrp.Tolerance != 30 || urltestGrp.URL != "https://www.google.com/generate_204" {
			t.Errorf("urltest group configuration mismatch: %+v", urltestGrp)
		}
		if len(urltestGrp.Outbounds) != 2 || urltestGrp.Outbounds[0] != "urltest_sec-out-1" || urltestGrp.Outbounds[1] != "urltest_sec-out-2" {
			t.Errorf("urltest group outbounds mismatch: %+v", urltestGrp.Outbounds)
		}
	}

	if selectorGrp == nil {
		t.Errorf("selector group not found")
	} else {
		if selectorGrp.Type != "selector" || selectorGrp.Default != "urltest_sec-urltest-out" {
			t.Errorf("selector group configuration mismatch: %+v", selectorGrp)
		}
		if len(selectorGrp.Outbounds) != 3 || selectorGrp.Outbounds[0] != "urltest_sec-urltest-out" || selectorGrp.Outbounds[1] != "urltest_sec-out-1" {
			t.Errorf("selector group outbounds mismatch: %+v", selectorGrp.Outbounds)
		}
	}

	if customOut == nil {
		t.Errorf("custom outbound not found")
	} else {
		if customOut.Type != "shadowsocks" || customOut.Server != "example.com" || customOut.ServerPort != 8388 || customOut.Method != "aes-256-gcm" || customOut.Password != "pwd" {
			t.Errorf("custom outbound configuration mismatch: %+v", customOut)
		}
	}
}
