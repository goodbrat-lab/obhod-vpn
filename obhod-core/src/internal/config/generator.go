package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/subscription"
	"net/url"
	"os"
	"strconv"
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

// sanitizePath cleans user input to prevent path traversal
func sanitizePath(filename string) string {
	// Remove path separators and dangerous characters
	reg := regexp.MustCompile(`[^\w\-\.]`)
	clean := reg.ReplaceAllString(filename, "")
	
	// Limit length
	if len(clean) > 32 {
		clean = clean[:32]
	}
	
	return clean
}

// safePath joins directory with sanitized filename
func safePath(dir, filename string) string {
	safeFilename := sanitizePath(filename)
	return filepath.Join(dir, safeFilename+".json")
}

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
			Rules:                 []RouteRuleConfig{},
			RuleSet:               []RuleSetConfig{},
			Final:                 "direct-out",
			AutoDetectInterface:   true,
			DefaultDomainResolver: "dns-proxy",
		},
		Experimental: &ExperimentalConfig{
			CacheFile: &CacheFileConfig{
				Path: uci.Settings.CachePath,
			},
			ClashAPI: &ClashAPIConfig{},
		},
	}

	// Always enable Clash API on 0.0.0.0:9090 for internal diagnostics and outbounds checks
	config.Experimental.ClashAPI.ExternalController = "0.0.0.0:9090"
	if uci.Settings.EnableYacd {
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

	// NOTE: In sing-box 1.12+, the 'dns' inbound type was removed.
	// DNS hijacking is now handled via a route rule with action: 'hijack-dns'.
	// The hijack-dns route rule is added in the route setup section below.

	// 2. DNS setup
	dnsStrategy := uci.Settings.DNSStrategy
	if dnsStrategy == "" {
		dnsStrategy = "ipv4_only"
	}
	config.DNS.Strategy = dnsStrategy
	setupDNS(config, uci)

	// 3. Outbounds & Route Rules
	config.Outbounds = append(config.Outbounds, OutboundConfig{
		Type:        "direct",
		Tag:         "direct-out",
		RoutingMark: 2097152, // 0x00200000 — bypass nftables tproxy rules
	})

	// Add sniff route rule first (required for hijack-dns to work - sing-box needs to detect DNS protocol)
	config.Route.Rules = append(config.Route.Rules, RouteRuleConfig{
		Action: "sniff",
	})

	// Add hijack-dns route rule (sing-box 1.12+ replacement for dns inbound)
	// IMPORTANT: Only intercept DNS from tproxy-in (client traffic).
	// Without this restriction, sing-box's own outbound DNS queries (e.g. resolving
	// the proxy server address via dns-direct) would also be hijacked, creating a loop.
	config.Route.Rules = append(config.Route.Rules, RouteRuleConfig{
		Inbound:  []string{"tproxy-in"},
		Protocol: []string{"dns"},
		Action:   "hijack-dns",
	})

	fetcher := subscription.NewFetcher("")
	if uci.Settings.DownloadListsViaProxy {
		secName := uci.Settings.DownloadListsViaProxySection
		if secName != "" {
			if sec, ok := uci.Sections[secName]; ok && sec.MixedProxyEnabled && sec.MixedProxyPort > 0 {
				fetcher.ProxyPort = sec.MixedProxyPort
			}
		} else {
			for _, sec := range uci.Sections {
				if sec.MixedProxyEnabled && sec.MixedProxyPort > 0 {
					fetcher.ProxyPort = sec.MixedProxyPort
					break
				}
			}
		}
	}
	cache, _ := fetcher.LoadCache()

	for _, section := range uci.Sections {
		if !section.Enabled {
			continue
		}
		processSection(config, section, fetcher, cache)
	}

	return config, nil
}

func getFirstProxyOutboundTag(uci *UCIConfig) string {
	if uci.Settings.DownloadListsViaProxySection != "" {
		if sec, ok := uci.Sections[uci.Settings.DownloadListsViaProxySection]; ok && sec.Enabled && sec.ConnectionType == "proxy" {
			return sec.Name + "-out"
		}
	}
	for _, sec := range uci.Sections {
		if sec.Enabled && sec.ConnectionType == "proxy" {
			return sec.Name + "-out"
		}
	}
	return ""
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

	// Set detour if a proxy outbound is available
	firstProxyTag := getFirstProxyOutboundTag(uci)
	if firstProxyTag != "" {
		server.Detour = firstProxyTag
	}

	// Set domain resolver if host is not an IPv4 address
	dnsServerHost := dnsServer
	if strings.Contains(dnsServer, "://") {
		if u, err := url.Parse(dnsServer); err == nil {
			dnsServerHost = u.Hostname()
		}
	} else if strings.Contains(dnsServer, "/") {
		parts := strings.SplitN(dnsServer, "/", 2)
		dnsServerHost = parts[0]
	}
	if h, _, err := net.SplitHostPort(dnsServerHost); err == nil {
		dnsServerHost = h
	}
	ip := net.ParseIP(dnsServerHost)
	if ip == nil || ip.To4() == nil {
		server.DomainResolver = "dns-direct"
	}

	config.DNS.Servers = append(config.DNS.Servers, server)

	// 3. FakeIP DNS (1.12+ format)
	config.DNS.Servers = append(config.DNS.Servers, DNSServerConfig{
		Type:       "fakeip",
		Tag:        "fakeip-server",
		Inet4Range: "198.18.0.0/15",
	})
	
	config.DNS.Final = mainTag
	
	// 4. DNS Rules
	config.DNS.Rules = append(config.DNS.Rules, DNSRuleConfig{
		Outbound: []string{"any"},
		Server:   "dns-direct",
	})
	
	rewriteTTL := uci.Settings.DNSRewriteTTL
	if rewriteTTL <= 0 {
		rewriteTTL = 60
	}
	config.DNS.Rules = append(config.DNS.Rules, DNSRuleConfig{
		Domain:     []string{"fakeip.podkop.fyi", "ip.podkop.fyi"},
		Server:     "fakeip-server",
		RewriteTTL: rewriteTTL,
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

		if section.EnableUDPOverTCP {
			for i := range outbounds {
				if outbounds[i].Type == "shadowsocks" || outbounds[i].Type == "socks" {
					outbounds[i].UdpOverTcp = map[string]interface{}{
						"enabled": true,
						"version": 2,
					}
				}
			}
		}

		if len(outbounds) == 0 {
			logger.Warn("config", "generator", "No valid proxies found for section %s. Creating a fallback direct outbound to prevent startup failure.", section.Name)
			outbounds = append(outbounds, OutboundConfig{
				Type:        "direct",
				Tag:         outboundTag,
				RoutingMark: 2097152,
			})
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
	// Sanitize: trim whitespace and remove stray \r\n from Windows or UCI quirks
	proxyStr = strings.TrimSpace(proxyStr)
	proxyStr = strings.ReplaceAll(proxyStr, "\r", "")
	proxyStr = strings.ReplaceAll(proxyStr, "\n", "")

	if proxyStr == "" {
		return nil, fmt.Errorf("empty proxy URL")
	}
	if strings.HasPrefix(proxyStr, "vmess://") {
		return parseVMess(proxyStr, tag)
	}

	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	outbound := &OutboundConfig{
		Tag:         tag,
		RoutingMark: 2097152, // 0x00200000 — bypass nftables tproxy rules
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
		if pe := u.Query().Get("packetEncoding"); pe != "" {
			outbound.PacketEncoding = pe
		} else if pe := u.Query().Get("packet_encoding"); pe != "" {
			outbound.PacketEncoding = pe
		}
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
		if decoded, err := DecodeBase64Tolerant(userPass); err == nil {
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

	case "socks", "socks5", "socks4", "socks4a":
		outbound.Type = "socks"
		outbound.Server = u.Hostname()
		outbound.ServerPort = 1080
		if u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &outbound.ServerPort)
		}
		if u.User != nil {
			outbound.Username = u.User.Username()
			outbound.Password, _ = u.User.Password()
		}
		outbound.Version = "5"
		if u.Scheme == "socks4" || u.Scheme == "socks4a" {
			outbound.Version = "4"
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
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
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
		// Reality TLS — requires public_key (pbk) and short_id (sid)
		if security == "reality" {
			pbk := u.Query().Get("pbk")
			sid := u.Query().Get("sid")
			if pbk != "" {
				outbound.TLS.Reality = &RealityConfig{
					Enabled:   true,
					PublicKey: pbk,
					ShortID:   sid,
				}
			}
		}
	}
}

func applyTransport(outbound *OutboundConfig, u *url.URL) {
	tType := u.Query().Get("type")
	switch tType {
	case "ws":
		outbound.Transport = &TransportConfig{
			Type: "ws",
			Path: u.Query().Get("path"),
		}
		if host := u.Query().Get("host"); host != "" {
			outbound.Transport.Host = []string{host}
		}
		// WebSocket 0-RTT early data
		if ed := u.Query().Get("ed"); ed != "" {
			if edInt, err := strconv.Atoi(ed); err == nil && edInt > 0 {
				outbound.Transport.MaxEarlyData = edInt
				outbound.Transport.EarlyDataHeaderName = "Sec-WebSocket-Protocol"
			}
		}
	case "grpc":
		outbound.Transport = &TransportConfig{
			Type:        "grpc",
			ServiceName: u.Query().Get("serviceName"),
		}
	case "http", "h2":
		outbound.Transport = &TransportConfig{
			Type: "http",
			Path: u.Query().Get("path"),
		}
		if host := u.Query().Get("host"); host != "" {
			outbound.Transport.Host = []string{host}
		}
	case "tcp", "raw", "":
		// No transport needed
	}
}

func parseVMess(proxyStr string, tag string) (*OutboundConfig, error) {
	data := strings.TrimPrefix(proxyStr, "vmess://")
	decoded, err := DecodeBase64Tolerant(data)
	if err == nil {
		var v map[string]interface{}
		if err := json.Unmarshal(decoded, &v); err == nil {
			outbound := &OutboundConfig{
				Type:        "vmess",
				Tag:         tag,
				Server:      fmt.Sprintf("%v", v["add"]),
				UUID:        fmt.Sprintf("%v", v["id"]),
				Security:    "auto",
				RoutingMark: 2097152,
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
	}

	// Fallback to URL format: vmess://uuid@host:port?security=auto&packetEncoding=x...
	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vmess: not base64 and not valid url: %v", err)
	}
	outbound := &OutboundConfig{
		Type:        "vmess",
		Tag:         tag,
		Server:      u.Hostname(),
		ServerPort:  443,
		UUID:        u.User.String(),
		Security:    "auto",
		RoutingMark: 2097152,
	}
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &outbound.ServerPort)
	}
	if sec := u.Query().Get("security"); sec != "" {
		outbound.Security = sec
	}
	if pe := u.Query().Get("packetEncoding"); pe != "" {
		outbound.PacketEncoding = pe
	} else if pe := u.Query().Get("packet_encoding"); pe != "" {
		outbound.PacketEncoding = pe
	}
	applyTLS(outbound, u)
	applyTransport(outbound, u)
	return outbound, nil
}

// DecodeBase64Tolerant decodes Base64 strings, handling URL-safe formats, padding, and whitespace
func DecodeBase64Tolerant(str string) ([]byte, error) {
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	str = strings.ReplaceAll(str, "\t", "")
	str = strings.ReplaceAll(str, " ", "")

	str = strings.ReplaceAll(str, "-", "+")
	str = strings.ReplaceAll(str, "_", "/")

	mod := len(str) % 4
	if mod == 2 {
		str += "=="
	} else if mod == 3 {
		str += "="
	}

	if data, err := base64.StdEncoding.DecodeString(str); err == nil {
		return data, nil
	}

	return base64.URLEncoding.DecodeString(str)
}
