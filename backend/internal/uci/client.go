package uci

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type GlobalSettings struct {
	Enabled    bool
	Fwmark     int
	DnsPort    int
	TproxyPort int
	LogLevel   string
}

type Tunnel struct {
	ID          string
	Type        string // vless, wireguard
	Server      string
	Port        int
	UUID        string // for vless
	Transport   string // tcp, ws, grpc
	Security    string // none, tls, reality
	Flow        string
	SNI         string
	Fingerprint string
	PublicKey   string
	ShortID     string
	SpiderX     string
	Path        string
	Host        string
}

type RoutingRule struct {
	ID      string
	Target  string
	Domains []string
}

type UciConfig struct {
	Settings GlobalSettings
	Tunnels  []Tunnel
	Routing  []RoutingRule
}

// executeUciShow is a variable so we can mock it in tests
var executeUciShow = func() ([]byte, error) {
	cmd := exec.Command("uci", "show", "obhod")
	return cmd.Output()
}

func LoadConfig() (*UciConfig, error) {
	out, err := executeUciShow()
	if err != nil {
		return nil, fmt.Errorf("failed to run uci show obhod: %w", err)
	}

	return parseUciShow(out)
}

func parseUciShow(data []byte) (*UciConfig, error) {
	cfg := &UciConfig{
		Settings: GlobalSettings{
			Enabled:    true,
			Fwmark:     255,
			DnsPort:    15353,
			TproxyPort: 11080,
			LogLevel:   "info",
		},
	}

	tunnelsMap := make(map[string]*Tunnel)
	rulesMap := make(map[string]*RoutingRule)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "obhod.") {
			continue
		}

		// e.g. obhod.settings.fwmark='255'
		// or obhod.main=tunnel
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		keyPart := parts[0]
		valPart := strings.Trim(parts[1], "'\"") // simple unquote

		keySegments := strings.Split(keyPart, ".")
		if len(keySegments) < 2 {
			continue
		}

		sectionID := keySegments[1]

		// It's a type definition, e.g., obhod.main=tunnel
		if len(keySegments) == 2 {
			if valPart == "tunnel" {
				tunnelsMap[sectionID] = &Tunnel{ID: sectionID}
			} else if valPart == "routing" {
				rulesMap[sectionID] = &RoutingRule{ID: sectionID}
			}
			continue
		}

		// Option or List, e.g. obhod.main.server='proxy.example.com'
		option := keySegments[2]

		if sectionID == "settings" || sectionID == "@global[0]" {
			switch option {
			case "enabled":
				cfg.Settings.Enabled = valPart == "1" || valPart == "true"
			case "fwmark":
				if v, err := strconv.Atoi(valPart); err == nil {
					cfg.Settings.Fwmark = v
				}
			case "dns_port":
				if v, err := strconv.Atoi(valPart); err == nil {
					cfg.Settings.DnsPort = v
				}
			case "tproxy_port":
				if v, err := strconv.Atoi(valPart); err == nil {
					cfg.Settings.TproxyPort = v
				}
			case "log_level":
				cfg.Settings.LogLevel = valPart
			}
		} else if t, ok := tunnelsMap[sectionID]; ok {
			switch option {
			case "type":
				t.Type = valPart
			case "server":
				t.Server = valPart
			case "port":
				if v, err := strconv.Atoi(valPart); err == nil {
					t.Port = v
				}
			case "uuid":
				t.UUID = valPart
			case "transport":
				t.Transport = valPart
			case "security":
				t.Security = valPart
			case "flow":
				t.Flow = valPart
			case "sni":
				t.SNI = valPart
			case "fingerprint":
				t.Fingerprint = valPart
			case "public_key":
				t.PublicKey = valPart
			case "short_id":
				t.ShortID = valPart
			case "spider_x":
				t.SpiderX = valPart
			case "path":
				t.Path = valPart
			case "host":
				t.Host = valPart
			}
		} else if r, ok := rulesMap[sectionID]; ok {
			switch option {
			case "target":
				r.Target = valPart
			case "domains":
				// Handle potential list format: 'val1' 'val2' or val1 val2
				parts := strings.Fields(valPart)
				for _, p := range parts {
					r.Domains = append(r.Domains, strings.Trim(p, "'\""))
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for _, t := range tunnelsMap {
		cfg.Tunnels = append(cfg.Tunnels, *t)
	}
	for _, r := range rulesMap {
		cfg.Routing = append(cfg.Routing, *r)
	}

	return cfg, nil
}
