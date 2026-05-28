package config

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

type UCIConfig struct {
	Settings SettingsUCI
	Sections map[string]SectionUCI
}

type SettingsUCI struct {
	Enabled                  bool
	LogLevel                 string
	UpdateInterval           string
	DontTouchDHCP            bool
	ConfigPath               string
	SourceNetworkInterfaces []string
	ExcludeNTP               bool
	CachePath                string
	WatchdogInterval         string
	DNSType                  string
	DNSServer                string
	DNSStrategy              string
	BootstrapDNSServer       string
	EnableYacd               bool
	EnableYacdWanAccess      bool
	YacdSecretKey            string
	TelegramToken            string
	TelegramChatID           string
	TelegramEnabled          bool
	DownloadListsViaProxy        bool
	DownloadListsViaProxySection string
	SingBoxVersion           string
	UseLegacyGenerator       bool
}

type SectionUCI struct {
	Name                    string
	Enabled                 bool
	ConnectionType          string
	ProxyConfigType         string
	ProxyString             string
	SubscriptionURL         string
	CommunityLists          []string
	UserDomains             []string
	UserSubnets             []string
	MixedProxyEnabled       bool
	MixedProxyPort          int
	SelectorProxyLinks      []string
}

func LoadUCI() (*UCIConfig, error) {
	cmd := exec.Command("uci", "-q", "show", "obhod")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	config := &UCIConfig{
		Sections: make(map[string]SectionUCI),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], "'")

		// obhod.settings.xxx or obhod.NAME.xxx
		keyParts := strings.Split(key, ".")
		if len(keyParts) < 2 {
			continue
		}

		sectionName := keyParts[1]

		if sectionName == "settings" {
			parseSettings(&config.Settings, keyParts, value)
		} else {
			section, ok := config.Sections[sectionName]
			if !ok {
				section = SectionUCI{Name: sectionName}
			}
			parseSection(&section, keyParts, value)
			config.Sections[sectionName] = section
		}
	}

	return config, nil
}

func parseSettings(s *SettingsUCI, keyParts []string, value string) {
	if len(keyParts) < 3 {
		return
	}
	option := keyParts[2]
	switch option {
	case "enabled":
		s.Enabled = value == "1"
	case "log_level":
		s.LogLevel = value
	case "update_interval":
		s.UpdateInterval = value
	case "dont_touch_dhcp":
		s.DontTouchDHCP = value == "1"
	case "config_path":
		s.ConfigPath = value
	case "source_network_interfaces":
		s.SourceNetworkInterfaces = append(s.SourceNetworkInterfaces, strings.Fields(value)...)
	case "exclude_ntp":
		s.ExcludeNTP = value == "1"
	case "cache_path":
		s.CachePath = value
	case "watchdog_interval":
		s.WatchdogInterval = value
	case "dns_type":
		s.DNSType = value
	case "dns_server":
		s.DNSServer = value
	case "dns_strategy":
		s.DNSStrategy = value
	case "bootstrap_dns_server":
		s.BootstrapDNSServer = value
	case "enable_yacd":
		s.EnableYacd = value == "1"
	case "enable_yacd_wan_access":
		s.EnableYacdWanAccess = value == "1"
	case "yacd_secret_key":
		s.YacdSecretKey = value
	case "telegram_token":
		s.TelegramToken = value
	case "telegram_chat_id":
		s.TelegramChatID = value
	case "telegram_enabled":
		s.TelegramEnabled = value == "1"
	case "download_lists_via_proxy":
		s.DownloadListsViaProxy = value == "1"
	case "download_lists_via_proxy_section":
		s.DownloadListsViaProxySection = value
	case "singbox_version":
		s.SingBoxVersion = value
	case "use_legacy_generator":
		s.UseLegacyGenerator = value == "1"
	}
}

func parseSection(s *SectionUCI, keyParts []string, value string) {
	if len(keyParts) < 3 {
		return
	}
	option := keyParts[2]
	switch option {
	case "enabled":
		s.Enabled = value == "1"
	case "connection_type":
		s.ConnectionType = value
	case "proxy_config_type":
		s.ProxyConfigType = value
	case "proxy_string":
		s.ProxyString = value
	case "subscription_url":
		s.SubscriptionURL = value
	case "community_lists":
		s.CommunityLists = append(s.CommunityLists, value)
	case "user_domains":
		s.UserDomains = append(s.UserDomains, value)
	case "user_subnets":
		s.UserSubnets = append(s.UserSubnets, value)
	case "mixed_proxy_enabled":
		s.MixedProxyEnabled = value == "1"
	case "mixed_proxy_port":
		if p, err := strconv.Atoi(value); err == nil {
			s.MixedProxyPort = p
		}
	case "selector_proxy_links":
		s.SelectorProxyLinks = append(s.SelectorProxyLinks, value)
	}
}
