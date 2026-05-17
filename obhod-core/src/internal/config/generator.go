package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/subscription"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Mapping of community list keys to their respective SRS URLs (Binary format)
var communityListMap = map[string]string{
	"twitter":      "https://github.com/itdoginfo/allow-domains/releases/latest/download/twitter.srs",
	"meta":         "https://github.com/itdoginfo/allow-domains/releases/latest/download/meta.srs",
	"telegram":     "https://github.com/itdoginfo/allow-domains/releases/latest/download/telegram.srs",
	"cloudflare":   "https://github.com/itdoginfo/allow-domains/releases/latest/download/cloudflare.srs",
	"hetzner":      "https://github.com/itdoginfo/allow-domains/releases/latest/download/hetzner.srs",
	"ovh":          "https://github.com/itdoginfo/allow-domains/releases/latest/download/ovh.srs",
	"digitalocean": "https://github.com/itdoginfo/allow-domains/releases/latest/download/digitalocean.srs",
	"cloudfront":   "https://github.com/itdoginfo/allow-domains/releases/latest/download/cloudfront.srs",
	"discord":      "https://github.com/itdoginfo/allow-domains/releases/latest/download/discord.srs",
	"roblox":       "https://github.com/itdoginfo/allow-domains/releases/latest/download/roblox.srs",
	"youtube":      "https://github.com/itdoginfo/allow-domains/releases/latest/download/youtube.srs",
	"google_ai":    "https://github.com/itdoginfo/allow-domains/releases/latest/download/google_ai.srs",
}

const (
	RulesDir = "/tmp/obhod/rules"
)

func Generate(uci *UCIConfig) (*SingBoxConfig, error) {
	// Ensure rules directory exists
	os.MkdirAll(RulesDir, 0755)

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
			Strategy: uci.Settings.DNSStrategy,
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
			ClashAPI: &ClashAPIConfig{},
		},
	}

	if uci.Settings.EnableYacd {
		config.Experimental.ClashAPI.ExternalController = "127.0.0.1:9090"
		if uci.Settings.EnableYacdWanAccess {
			config.Experimental.ClashAPI.ExternalController = "0.0.0.0:9090"
		}
		config.Experimental.ClashAPI.Secret = uci.Settings.YacdSecretKey
		config.Experimental.ClashAPI.ExternalUI = "ui"
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

	config.Inbounds = append(config.Inbounds, InboundConfig{
		Type:       "dns",
		Tag:        "dns-in",
		Listen:     "127.0.0.42",
		ListenPort: 53,
	})

	// 2. DNS setup
	dnsStrategy := uci.Settings.DNSStrategy
	if dnsStrategy == "" {
		dnsStrategy = "ipv4_only"
	}
	config.DNS.Strategy = dnsStrategy
	setupDNS(config, uci)

	// 3. Outbounds & Route Rules
	config.Outbounds = append(config.Outbounds, OutboundConfig{
		Type: "direct",
		Tag:  "direct-out",
	})

	fetcher := subscription.NewFetcher("")
	cache, _ := fetcher.LoadCache()

	for _, section := range uci.Sections {
		if !section.Enabled {
			continue
		}
		processSection(config, section, fetcher, cache)
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

	// 1. Direct DNS (Bootstrap)
	config.DNS.Servers = append(config.DNS.Servers, DNSServerConfig{
		Type:   "udp",
		Tag:    "dns-direct",
		Server: bootstrapServer,
		Detour: "direct-out",
	})

	// 2. Default Tunnel DNS
	mainTag := "dns-proxy"
	dnsType := uci.Settings.DNSType
	if dnsType == "" {
		dnsType = "udp"
	}

	server := DNSServerConfig{
		Tag:    mainTag,
		Server: dnsServer,
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

	// 3. FakeIP DNS
	config.DNS.Servers = append(config.DNS.Servers, DNSServerConfig{
		Tag:     "fakeip-server",
		Address: "fakeip",
	})
	
	config.DNS.FakeIP = &DNSFakeIPConfig{
		Enabled:    true,
		Inet4Range: "198.18.0.0/15",
	}
	if config.Experimental.CacheFile != nil {
		config.Experimental.CacheFile.StoreFakeIP = true
	}
	
	config.DNS.Final = mainTag
	
	// 4. DNS Rules
	config.DNS.Rules = append(config.DNS.Rules, DNSRuleConfig{
		Outbound: []string{"any"},
		Server:   "dns-direct",
	})
	
	// Reject known DoH/DoT probing domains or specific types by using block (or direct as fallback).
	// In sing-box, we can just let them go direct or block. We'll use dns-direct for proxy internal resolution.
	// But we need to make sure fakeip rule-sets route to fakeip-server!
	// (The generator automatically adds community rulesets to use section DNS server which resolves them.
	// Actually, the bash script routed fakeip-dns-rule-tag to fakeip-server.)
	config.DNS.Rules = append(config.DNS.Rules, DNSRuleConfig{
		Domain:     []string{"fakeip.podkop.fyi", "ip.podkop.fyi"},
		Server:     "fakeip-server",
		RewriteTTL: 60,
	})
}

func processSection(config *SingBoxConfig, section SectionUCI, fetcher *subscription.Fetcher, cache *subscription.CacheData) {
	outboundTag := section.Name + "-out"

	if section.ConnectionType == "proxy" {
		var outbounds []OutboundConfig

		switch section.ProxyConfigType {
		case "url":
			outbound, err := parseProxyURL(section.ProxyString, outboundTag)
			if err == nil && outbound != nil {
				outbounds = append(outbounds, *outbound)
			}
		case "subscription":
			if section.SubscriptionURL != "" {
				var links []string
				if cache != nil {
					if cachedLinks, ok := cache.Sections[section.Name]; ok {
						links = cachedLinks
					}
				}
				if len(links) == 0 {
					var err error
					links, err = fetcher.Fetch(section.SubscriptionURL)
					if err != nil {
						logger.Error("config", "generator", "Failed to fetch subscription for %s: %v", section.Name, err)
					}
				}

				for i, link := range links {
					tag := fmt.Sprintf("%s-%d", outboundTag, i)
					outbound, err := parseProxyURL(link, tag)
					if err == nil && outbound != nil {
						outbounds = append(outbounds, *outbound)
					}
				}
			}
		case "selector":
			for i, link := range section.SelectorProxyLinks {
				tag := fmt.Sprintf("%s-%d", outboundTag, i+1)
				outbound, err := parseProxyURL(link, tag)
				if err == nil && outbound != nil {
					outbounds = append(outbounds, *outbound)
				}
			}
		}

		if len(outbounds) > 0 {
			finalOutboundTag := outboundTag

			if len(outbounds) > 1 || section.ProxyConfigType == "selector" {
				var tags []string
				for _, o := range outbounds {
					config.Outbounds = append(config.Outbounds, o)
					tags = append(tags, o.Tag)
				}

				groupType := "urltest"
				if section.ProxyConfigType == "selector" {
					groupType = "selector"
				}

				group := OutboundConfig{
					Type:      groupType,
					Tag:       outboundTag,
					Outbounds: tags,
				}
				config.Outbounds = append(config.Outbounds, group)
			} else {
				config.Outbounds = append(config.Outbounds, outbounds[0])
			}

			sectionDNSTag := "dns-" + section.Name
			config.DNS.Servers = append(config.DNS.Servers, DNSServerConfig{
				Type:   "udp",
				Tag:    sectionDNSTag,
				Server: "8.8.8.8",
				Detour: finalOutboundTag,
			})

			// 1. Community Lists
			for _, service := range section.CommunityLists {
				rulesetTag := "rs-" + service

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
						Format:         "binary",
						URL:            getCommunityURL(service),
						UpdateInterval: "1d",
					}
					config.Route.RuleSet = append(config.Route.RuleSet, rs)
				}

				config.DNS.Rules = append(config.DNS.Rules, DNSRuleConfig{
					RuleSet: []string{rulesetTag},
					Server:  "fakeip-server",
				})

				rule := RouteRuleConfig{
					Inbound:  []string{"tproxy-in"},
					RuleSet:  []string{rulesetTag},
					Outbound: finalOutboundTag,
				}
				config.Route.Rules = append(config.Route.Rules, rule)
			}

			// 2. User Domains (External Rule-set)
			if len(section.UserDomains) > 0 {
				userTag := "user-domains-" + section.Name
				path := filepath.Join(RulesDir, userTag+".json")
				
				// RAM Optimization: Write to external file
				ruleData := map[string]interface{}{
					"version": 1,
					"rules": []map[string]interface{}{
						{
							"domain": section.UserDomains,
						},
					},
				}
				if file, err := json.Marshal(ruleData); err == nil {
					os.WriteFile(path, file, 0644)
				}

				rs := RuleSetConfig{
					Type:   "local",
					Tag:    userTag,
					Format: "source",
					Path:   path,
				}
				config.Route.RuleSet = append(config.Route.RuleSet, rs)

				config.DNS.Rules = append(config.DNS.Rules, DNSRuleConfig{
					RuleSet: []string{userTag},
					Server:  "fakeip-server",
				})

				config.Route.Rules = append(config.Route.Rules, RouteRuleConfig{
					Inbound:  []string{"tproxy-in"},
					RuleSet:  []string{userTag},
					Outbound: finalOutboundTag,
				})
			}

			// 3. User Subnets (External Rule-set)
			if len(section.UserSubnets) > 0 {
				userSubnetTag := "user-subnets-" + section.Name
				path := filepath.Join(RulesDir, userSubnetTag+".json")
				
				// RAM Optimization: Write to external file
				ruleData := map[string]interface{}{
					"version": 1,
					"rules": []map[string]interface{}{
						{
							"ip_cidr": section.UserSubnets,
						},
					},
				}
				if file, err := json.Marshal(ruleData); err == nil {
					os.WriteFile(path, file, 0644)
				}

				rs := RuleSetConfig{
					Type:   "local",
					Tag:    userSubnetTag,
					Format: "source",
					Path:   path,
				}
				config.Route.RuleSet = append(config.Route.RuleSet, rs)

				config.Route.Rules = append(config.Route.Rules, RouteRuleConfig{
					Inbound:  []string{"tproxy-in"},
					RuleSet:  []string{userSubnetTag},
					Outbound: finalOutboundTag,
				})
			}
		}
	}
}

func getCommunityURL(service string) string {
	if url, ok := communityListMap[service]; ok {
		return url
	}
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
