package config

type SingBoxConfig struct {
	Log          *LogConfig          `json:"log,omitempty"`
	DNS          *DNSConfig          `json:"dns,omitempty"`
	Inbounds     []InboundConfig     `json:"inbounds,omitempty"`
	Outbounds    []OutboundConfig    `json:"outbounds,omitempty"`
	Route        *RouteConfig        `json:"route,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
}

type LogConfig struct {
	Disabled  bool   `json:"disabled,omitempty"`
	Level     string `json:"level,omitempty"`
	Timestamp bool   `json:"timestamp,omitempty"`
}

type DNSConfig struct {
	Servers          []DNSServerConfig `json:"servers,omitempty"`
	Rules            []DNSRuleConfig   `json:"rules,omitempty"`
	Final            string            `json:"final,omitempty"`
	Strategy         string            `json:"strategy,omitempty"`
	IndependentCache bool              `json:"independent_cache,omitempty"`
	ReverseMapping   bool              `json:"reverse_mapping,omitempty"`
}

type DNSServerConfig struct {
	Type            string   `json:"type,omitempty"`
	Tag             string   `json:"tag,omitempty"`
	Server          string   `json:"server,omitempty"`
	Address         string   `json:"address,omitempty"`
	ServerPort      int      `json:"server_port,omitempty"`
	Inet4Range      string   `json:"inet4_range,omitempty"`
	Inet6Range      string   `json:"inet6_range,omitempty"`
	DomainResolver  string   `json:"domain_resolver,omitempty"`
	DomainStrategy  string   `json:"domain_strategy,omitempty"`
	Detour          string   `json:"detour,omitempty"`
}

type DNSRuleConfig struct {
	Inbound      []string `json:"inbound,omitempty"`
	Outbound     []string `json:"outbound,omitempty"`
	RuleSet      []string `json:"rule_set,omitempty"`
	Server       string   `json:"server,omitempty"`
	DisableCache bool     `json:"disable_cache,omitempty"`
	QueryType    []string `json:"query_type,omitempty"`
	DomainSuffix []string `json:"domain_suffix,omitempty"`
	Domain       []string `json:"domain,omitempty"`
	RewriteTTL   int      `json:"rewrite_ttl,omitempty"`
}

type InboundConfig struct {
	Type              string `json:"type"`
	Tag               string `json:"tag,omitempty"`
	Listen            string `json:"listen,omitempty"`
	ListenPort        int    `json:"listen_port,omitempty"`
	Sniff             bool   `json:"sniff,omitempty"`
	SniffOverrideDestination bool `json:"sniff_override_destination,omitempty"`
}

type OutboundConfig struct {
	Type string `json:"type"`
	Tag  string `json:"tag,omitempty"`

	// Group fields
	Outbounds []string `json:"outbounds,omitempty"`
	Default   string   `json:"default,omitempty"`

	// Common Proxy fields
	Server     string `json:"server,omitempty"`
	ServerPort int    `json:"server_port,omitempty"`

	// Shadowsocks / SOCKS
	Method   string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
	Version  string `json:"version,omitempty"`
	Username string `json:"username,omitempty"`
	Network  string `json:"network,omitempty"`
	UdpOverTcp interface{} `json:"udp_over_tcp,omitempty"`

	// VLESS / VMess / Trojan
	UUID           string `json:"uuid,omitempty"`
	Flow           string `json:"flow,omitempty"`
	Security       string `json:"security,omitempty"`
	AlterId        int    `json:"alter_id,omitempty"`
	PacketEncoding string `json:"packet_encoding,omitempty"`

	// Hysteria2
	UpMbps   int         `json:"up_mbps,omitempty"`
	DownMbps int         `json:"down_mbps,omitempty"`
	Obfs     *ObfsConfig `json:"obfs,omitempty"`

	// Transport & TLS
	TLS       *TLSConfig       `json:"tls,omitempty"`
	Transport *TransportConfig `json:"transport,omitempty"`

	// Multiplex
	Multiplex *MultiplexConfig `json:"multiplex,omitempty"`
}

type TLSConfig struct {
	Enabled    bool        `json:"enabled,omitempty"`
	ServerName string      `json:"server_name,omitempty"`
	Insecure   bool        `json:"insecure,omitempty"`
	Alpn       []string    `json:"alpn,omitempty"`
	UTLS       *UTLSConfig `json:"utls,omitempty"`
}

type UTLSConfig struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type TransportConfig struct {
	Type                string            `json:"type"`
	Path                string            `json:"path,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	Host                []string          `json:"host,omitempty"`
	ServiceName         string            `json:"service_name,omitempty"`
	EarlyDataHeaderName string            `json:"early_data_header_name,omitempty"`
}

type ObfsConfig struct {
	Type     string `json:"type"`
	Password string `json:"password,omitempty"`
}

type MultiplexConfig struct {
	Enabled    bool   `json:"enabled,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	MaxStreams int    `json:"max_streams,omitempty"`
}

type RouteConfig struct {
	Rules               []RouteRuleConfig `json:"rules,omitempty"`
	RuleSet             []RuleSetConfig   `json:"rule_set,omitempty"`
	Final               string            `json:"final,omitempty"`
	AutoDetectInterface bool              `json:"auto_detect_interface,omitempty"`
}

type RuleSetConfig struct {
	Type           string      `json:"type"`
	Tag            string      `json:"tag"`
	Format         string      `json:"format,omitempty"`
	Path           string      `json:"path,omitempty"`
	URL            string      `json:"url,omitempty"`
	DownloadDetour string      `json:"download_detour,omitempty"`
	UpdateInterval string      `json:"update_interval,omitempty"`
	Inline         interface{} `json:"inline,omitempty"`
}

type RouteRuleConfig struct {
	// Match conditions
	Inbound     []string `json:"inbound,omitempty"`
	IPCIDR      []string `json:"ip_cidr,omitempty"`
	RuleSet     []string `json:"rule_set,omitempty"`
	Port        []int    `json:"port,omitempty"`
	PortRange   []string `json:"port_range,omitempty"`
	Protocol    []string `json:"protocol,omitempty"`
	// Actions (sing-box 1.12+)
	// When Action is set, use action-based routing instead of Outbound
	Action   string `json:"action,omitempty"`
	Outbound string `json:"outbound,omitempty"`
}

type ExperimentalConfig struct {
	ClashAPI  *ClashAPIConfig  `json:"clash_api,omitempty"`
	CacheFile *CacheFileConfig `json:"cache_file,omitempty"`
}

type ClashAPIConfig struct {
	ExternalController string `json:"external_controller,omitempty"`
	ExternalUI         string `json:"external_ui,omitempty"`
	Secret             string `json:"secret,omitempty"`
}

type CacheFileConfig struct {
	Path     string `json:"path,omitempty"`
	CacheID  string `json:"cache_id,omitempty"`
	StoreFakeIP bool `json:"store_fakeip,omitempty"`
}
