package config

import (
	"encoding/json"
	"github.com/obhod/obhoud/internal/uci"
)

type SingBoxConfig struct {
	Log       Log        `json:"log"`
	DNS       DNS        `json:"dns"`
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
	Route     Route      `json:"route"`
}

type Log struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

type DNS struct {
	Servers []DNSServer `json:"servers"`
	Rules   []DNSRule   `json:"rules"`
	FakeIP  *FakeIP     `json:"fakeip,omitempty"`
}

type DNSServer struct {
	Tag     string `json:"tag"`
	Address string `json:"address"`
	Detour  string `json:"detour,omitempty"`
}

type DNSRule struct {
	DomainSuffix []string `json:"domain_suffix,omitempty"`
	Server       string   `json:"server"`
}

type FakeIP struct {
	Enabled    bool   `json:"enabled"`
	Inet4Range string `json:"inet4_range"`
}

type Inbound struct {
	Type          string `json:"type"`
	Tag           string `json:"tag"`
	Listen        string `json:"listen"`
	ListenPort    int    `json:"listen_port"`
	Sniff         bool   `json:"sniff,omitempty"`
	SniffOverride bool   `json:"sniff_override_destination,omitempty"`
}

type Outbound struct {
	Type   string `json:"type"`
	Tag    string `json:"tag"`
	Server string `json:"server,omitempty"`
	Port   int    `json:"server_port,omitempty"`
	UUID   string `json:"uuid,omitempty"`
	Flow   string `json:"flow,omitempty"`

	TLS       *TLSConfig       `json:"tls,omitempty"`
	Transport *TransportConfig `json:"transport,omitempty"`
}

type TLSConfig struct {
	Enabled    bool           `json:"enabled"`
	ServerName string         `json:"server_name,omitempty"`
	UTLS       *UTLSConfig    `json:"utls,omitempty"`
	Reality    *RealityConfig `json:"reality,omitempty"`
}

type UTLSConfig struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type RealityConfig struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key,omitempty"`
	ShortID   string `json:"short_id,omitempty"`
	SpiderX   string `json:"spider_x,omitempty"`
}

type TransportConfig struct {
	Type    string            `json:"type"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Route struct {
	Rules []RouteRule `json:"rules"`
}

type RouteRule struct {
	DomainSuffix []string `json:"domain_suffix,omitempty"`
	Outbound     string   `json:"outbound"`
}

func GenerateSingBoxConfig(cfg *uci.UciConfig) ([]byte, error) {
	var directDomains []string
	for _, t := range cfg.Tunnels {
		// If server is a domain name (not an IP), we must resolve it directly
		if t.Server != "" {
			directDomains = append(directDomains, t.Server)
		}
	}

	sb := SingBoxConfig{
		Log: Log{
			Level:     cfg.Settings.LogLevel,
			Timestamp: true,
		},
		DNS: DNS{
			Servers: []DNSServer{
				{
					Tag:     "dns-fakeip",
					Address: "fakeip",
				},
				{
					Tag:     "dns-remote",
					Address: "https://8.8.8.8/dns-query",
					Detour:  "direct", // fallback, typically detoured via proxy
				},
				{
					Tag:     "dns-direct",
					Address: "local",
					Detour:  "direct",
				},
			},
			Rules: []DNSRule{
				{
					DomainSuffix: []string{"lan", "local"},
					Server:       "dns-direct",
				},
			},
			FakeIP: &FakeIP{
				Enabled:    true,
				Inet4Range: "198.18.0.0/15",
			},
		},
		Inbounds: []Inbound{
			{
				Type:       "tproxy",
				Tag:        "tproxy-in",
				Listen:     "::",
				ListenPort: cfg.Settings.TproxyPort,
				Sniff:      true,
			},
			{
				Type:       "direct",
				Tag:        "dns-in",
				Listen:     "::",
				ListenPort: cfg.Settings.DnsPort,
			},
		},
	}

	if len(directDomains) > 0 {
		sb.DNS.Rules = append(sb.DNS.Rules, DNSRule{
			DomainSuffix: directDomains,
			Server:       "dns-direct",
		})
	}

	// Add Direct Outbound
	sb.Outbounds = append(sb.Outbounds, Outbound{
		Type: "direct",
		Tag:  "direct",
	})

	// Add Tunnels (Outbounds)
	for _, t := range cfg.Tunnels {
		ob := Outbound{
			Type:   t.Type,
			Tag:    "tunnel_" + t.ID,
			Server: t.Server,
			Port:   t.Port,
			UUID:   t.UUID,
			Flow:   t.Flow,
		}

		if t.Type == "vless" {
			if t.Security == "tls" || t.Security == "reality" {
				ob.TLS = &TLSConfig{
					Enabled:    true,
					ServerName: t.SNI,
				}
				if t.Fingerprint != "" {
					ob.TLS.UTLS = &UTLSConfig{
						Enabled:     true,
						Fingerprint: t.Fingerprint,
					}
				}
				if t.Security == "reality" {
					ob.TLS.Reality = &RealityConfig{
						Enabled:   true,
						PublicKey: t.PublicKey,
						ShortID:   t.ShortID,
						SpiderX:   t.SpiderX,
					}
				}
			}

			if t.Transport != "" && t.Transport != "tcp" {
				ob.Transport = &TransportConfig{
					Type: t.Transport,
					Path: t.Path,
				}
				if t.Host != "" {
					ob.Transport.Headers = map[string]string{
						"Host": t.Host,
					}
				}
			}
		}

		sb.Outbounds = append(sb.Outbounds, ob)
	}

	// Add Route Rules
	for _, r := range cfg.Routing {
		// In a real implementation, domains might refer to a geosite or list,
		// here we just map them directly as suffixes for illustration.
		rr := RouteRule{
			DomainSuffix: r.Domains,
			Outbound:     r.Target,
		}
		sb.Route.Rules = append(sb.Route.Rules, rr)
	}

	return json.MarshalIndent(sb, "", "  ")
}
