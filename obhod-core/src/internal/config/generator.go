package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
	"net/url"
	"strconv"
	"strings"
)

// Mapping of community list keys to their respective SRS URLs
var communityListMap = map[string]string{
	"twitter":      "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/twitter.lst",
	"meta":         "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/meta.lst",
	"telegram":     "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/telegram.lst",
	"cloudflare":   "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/cloudflare.lst",
	"hetzner":      "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/hetzner.lst",
	"ovh":          "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/ovh.lst",
	"digitalocean": "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/digitalocean.lst",
	"cloudfront":   "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/cloudfront.lst",
	"discord":      "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/discord.lst",
	"roblox":       "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Subnets/IPv4/roblox.lst",
}

func Generate(uci *UCIConfig) (*SingBoxConfig, error) {
	config := &SingBoxConfig{
		Log: &LogConfig{
			Level:     uci.Settings.LogLevel,
			Timestamp: true,
		},
		Inbounds:  []InboundConfig{},
		Outbounds: []OutboundConfig{},
		DNS: &DNSConfig{
			Servers: []DNSServerConfig{},
			Rules:   []DNSRuleConfig{},
			Final:   "direct-out",
		},
		Route: &RouteConfig{
			Rules:               []RouteRuleConfig{},
			RuleSet:             []RuleSetConfig{},
			Final:               "direct-out",
			AutoDetectInterface: true,
		},
		Experimental: &ExperimentalConfig{
			CacheFile: &CacheFileConfig{
				Path: uci.Settings.CachePath,
			},
		},
	}

	// 1. Inbounds
	config.Inbounds = append(config.Inbounds, InboundConfig{
		Type:                     "tproxy",
		Tag:                      "tproxy-in",
		Listen:                   "127.0.0.1",
		ListenPort:               1602,
		Sniff:                    true,
		SniffOverrideDestination: true,
	})

	// 2. DNS setup
	setupDNS(config, uci)

	// 3. Outbounds & Route Rules
	config.Outbounds = append(config.Outbounds, OutboundConfig{
		Type: "direct",
		Tag:  "direct-out",
	})

	for _, section := range uci.Sections {
		if !section.Enabled {
			continue
		}
		processSection(config, section)
	}

	return config, nil
}

func setupDNS(config *SingBoxConfig, uci *UCIConfig) {
	dnsServer := uci.Settings.DNSServer
	if dnsServer == "" {
		dnsServer = "8.8.8.8"
	}
	bootstrapServer := uci.Settings.BootstrapDNSServer
	if bootstrapServer == "" {
		bootstrapServer = "77.88.8.8"
	}

	// Bootstrap
	config.DNS.Servers = append(config.DNS.Servers, DNSServerConfig{
		Type:    "udp",
		Tag:     "bootstrap-dns-server",
		Address: bootstrapServer,
		Detour:  "direct-out",
	})

	// Main
	mainTag := "dns-server"
	dnsType := uci.Settings.DNSType
	if dnsType == "" {
		dnsType = "udp"
	}

	server := DNSServerConfig{
		Tag:     mainTag,
		Address: dnsServer,
		Detour:  "direct-out",
	}

	switch dnsType {
	case "doh":
		server.Type = "https"
	case "dot":
		server.Type = "tls"
	default:
		server.Type = "udp"
	}
	config.DNS.Servers = append(config.DNS.Servers, server)
	config.DNS.Final = mainTag
}

func processSection(config *SingBoxConfig, section SectionUCI) {
	outboundTag := section.Name + "-out"

	if section.ConnectionType == "proxy" {
		var outbound *OutboundConfig
		var err error

		switch section.ProxyConfigType {
		case "url":
			outbound, err = parseProxyURL(section.ProxyString, outboundTag)
		}

		if err != nil {
			logger.Error("config", "generator", "Section %s: %v", section.Name, err)
			return
		}

		if outbound != nil {
			config.Outbounds = append(config.Outbounds, *outbound)

			// Add route rules for community lists
			for _, service := range section.CommunityLists {
				rulesetTag := "rs-" + service
				
				// 1. Add rule-set definition if not already present
				exists := false
				for _, rs := range config.Route.RuleSet {
					if rs.Tag == rulesetTag {
						exists = true
						break
					}
				}

				if !exists {
					rs := RuleSetConfig{
						Type:           "remote",
						Tag:            rulesetTag,
						Format:         "source",
						URL:            getCommunityURL(service),
						UpdateInterval: "1d",
					}
					config.Route.RuleSet = append(config.Route.RuleSet, rs)
				}

				// 2. Add route rule
				rule := RouteRuleConfig{
					Inbound:  []string{"tproxy-in"},
					RuleSet:  []string{rulesetTag},
					Outbound: outboundTag,
				}
				config.Route.Rules = append(config.Route.Rules, rule)
			}
		}
	}
}

func getCommunityURL(service string) string {
	if url, ok := communityListMap[service]; ok {
		return url
	}
	// Fallback to a default structure if not in map
	return fmt.Sprintf("https://github.com/itdoginfo/allow-domains/releases/latest/download/%s.srs", service)
}

func parseProxyURL(proxyStr string, tag string) (*OutboundConfig, error) {
	if strings.HasPrefix(proxyStr, "vmess://") {
		return parseVMess(proxyStr, tag)
	}

	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	outbound := &OutboundConfig{
		Tag: tag,
	}

	switch u.Scheme {
	case "vless":
		outbound.Type = "vless"
		outbound.Server = u.Hostname()
		outbound.ServerPort = 443
		if u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &outbound.ServerPort)
		}
		outbound.UUID = u.User.String()
		outbound.Flow = u.Query().Get("flow")
		outbound.Security = u.Query().Get("security")
		applyTLS(outbound, u)
		applyTransport(outbound, u)

	case "ss":
		outbound.Type = "shadowsocks"
		outbound.Server = u.Hostname()
		outbound.ServerPort = 8388
		if u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &outbound.ServerPort)
		}
		userPass := u.User.String()
		if decoded, err := base64.StdEncoding.DecodeString(userPass); err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				outbound.Method = parts[0]
				outbound.Password = parts[1]
			}
		} else {
			parts := strings.SplitN(userPass, ":", 2)
			if len(parts) == 2 {
				outbound.Method = parts[0]
				outbound.Password = parts[1]
			}
		}

	case "trojan":
		outbound.Type = "trojan"
		outbound.Server = u.Hostname()
		outbound.ServerPort = 443
		if u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &outbound.ServerPort)
		}
		outbound.Password = u.User.String()
		applyTLS(outbound, u)
		applyTransport(outbound, u)

	case "hysteria2", "hy2":
		outbound.Type = "hysteria2"
		outbound.Server = u.Hostname()
		outbound.ServerPort = 443
		if u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &outbound.ServerPort)
		}
		outbound.Password = u.User.String()
		outbound.UpMbps, _ = strconv.Atoi(u.Query().Get("upmbps"))
		outbound.DownMbps, _ = strconv.Atoi(u.Query().Get("downmbps"))
		if obfs := u.Query().Get("obfs"); obfs != "" {
			outbound.Obfs = &ObfsConfig{
				Type:     obfs,
				Password: u.Query().Get("obfs-password"),
			}
		}
		applyTLS(outbound, u)
	}

	return outbound, nil
}

func applyTLS(outbound *OutboundConfig, u *url.URL) {
	security := u.Query().Get("security")
	if security == "" {
		if u.Scheme == "hysteria2" || u.Scheme == "hy2" {
			security = "tls"
		}
	}

	if security == "tls" || security == "reality" {
		outbound.TLS = &TLSConfig{
			Enabled:    true,
			ServerName: u.Query().Get("sni"),
			Insecure:   u.Query().Get("allowInsecure") == "1" || u.Query().Get("insecure") == "1",
		}
		if alpn := u.Query().Get("alpn"); alpn != "" {
			outbound.TLS.Alpn = strings.Split(alpn, ",")
		}
		if fp := u.Query().Get("fp"); fp != "" {
			outbound.TLS.UTLS = &UTLSConfig{
				Enabled:     true,
				Fingerprint: fp,
			}
		}
	}
}

func applyTransport(outbound *OutboundConfig, u *url.URL) {
	tType := u.Query().Get("type")
	if tType == "ws" {
		outbound.Transport = &TransportConfig{
			Type: "ws",
			Path: u.Query().Get("path"),
		}
		if host := u.Query().Get("host"); host != "" {
			outbound.Transport.Host = []string{host}
		}
	} else if tType == "grpc" {
		outbound.Transport = &TransportConfig{
			Type:        "grpc",
			ServiceName: u.Query().Get("serviceName"),
		}
	}
}

func parseVMess(proxyStr string, tag string) (*OutboundConfig, error) {
	data := strings.TrimPrefix(proxyStr, "vmess://")
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vmess base64: %v", err)
	}

	var v map[string]interface{}
	if err := json.Unmarshal(decoded, &v); err != nil {
		return nil, fmt.Errorf("failed to parse vmess json: %v", err)
	}

	outbound := &OutboundConfig{
		Type:     "vmess",
		Tag:      tag,
		Server:   fmt.Sprintf("%v", v["add"]),
		UUID:     fmt.Sprintf("%v", v["id"]),
		Security: "auto",
	}

	if port, ok := v["port"].(float64); ok {
		outbound.ServerPort = int(port)
	} else if portStr, ok := v["port"].(string); ok {
		fmt.Sscanf(portStr, "%d", &outbound.ServerPort)
	}

	if tls, ok := v["tls"].(string); ok && (tls == "tls" || tls == "reality") {
		outbound.TLS = &TLSConfig{
			Enabled:    true,
			ServerName: fmt.Sprintf("%v", v["sni"]),
		}
	}

	// Transport for VMess (V2RayN)
	if net, ok := v["net"].(string); ok {
		if net == "ws" {
			outbound.Transport = &TransportConfig{
				Type: "ws",
				Path: fmt.Sprintf("%v", v["path"]),
			}
			if host, ok := v["host"].(string); ok && host != "" {
				outbound.Transport.Host = []string{host}
			}
		} else if net == "grpc" {
			outbound.Transport = &TransportConfig{
				Type:        "grpc",
				ServiceName: fmt.Sprintf("%v", v["path"]),
			}
		}
	}

	return outbound, nil
}
