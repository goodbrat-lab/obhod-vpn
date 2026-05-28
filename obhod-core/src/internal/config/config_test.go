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
