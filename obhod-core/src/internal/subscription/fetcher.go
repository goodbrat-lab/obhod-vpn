package subscription

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/logger"
)

type Fetcher struct {
	Client    *http.Client
	CachePath string
	ProxyPort int
}

type CacheData struct {
	LastUpdate time.Time           `json:"last_update"`
	Sections   map[string][]string `json:"sections"`
}

const (
	// Maximum subscription size to prevent memory exhaustion (1MB)
	maxSubscriptionSize = 1024 * 1024
	// Maximum nodes per subscription to prevent memory exhaustion
	maxNodesPerSubscription = 500
	// Maximum line length to prevent DoS
	maxLineLength = 1000
)

// ValidateURL checks if URL is safe and well-formed
func validateURL(rawURL string) error {
	if len(rawURL) > maxLineLength {
		return fmt.Errorf("URL too long")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	return nil
}

func SanitizeURLForLog(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid-url]"
	}
	q := u.Query()
	sensitive := []string{"token", "key", "secret", "auth", "password", "tk", "uid"}
	for _, k := range sensitive {
		if _, exists := q[k]; exists {
			q.Set(k, "***")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func NewFetcher(cachePath string) *Fetcher {
	if cachePath == "" {
		cachePath = "/tmp/obhod/subscriptions.json"
	}
	return &Fetcher{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		CachePath: cachePath,
		ProxyPort: 0,
	}
}

func (f *Fetcher) Fetch(url string) ([]string, error) {
	// Validate URL first
	if err := validateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	
	var body []byte
	var httpCode int
	var err error

	// Determine if proxy should be used
	useProxy := f.ProxyPort > 0

	if useProxy {
		logger.Info("subscription", "fetch", "Attempting to download subscription via proxy (port %d)", f.ProxyPort)
		body, httpCode, err = f.fetchWithClient(url, f.ProxyPort, 5*time.Second)
		if err != nil || httpCode != http.StatusOK {
			logger.Warn("subscription", "fetch", "Failed to download subscription via proxy: %v (HTTP %d). Falling back to direct connection.", err, httpCode)
			body, httpCode, err = f.fetchWithClient(url, 0, 30*time.Second)
		}
	} else {
		logger.Debug("subscription", "fetch", "Downloading subscription directly...")
		body, httpCode, err = f.fetchWithClient(url, 0, 30*time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to download subscription: %v", err)
	}

	if httpCode != http.StatusOK {
		return nil, fmt.Errorf("subscription fetch returned HTTP %d", httpCode)
	}

	content := string(body)
	links := f.Parse(content)

	if len(links) == 0 {
		// Try Base64 decode (with trim to handle whitespace/newlines)
		trimmed := strings.TrimSpace(content)
		if len(trimmed) > 0 {
			decoded, err := DecodeBase64Tolerant(trimmed)
			if err == nil {
				// Limit decoded content size
				decodedStr := string(decoded)
				if int64(len(decodedStr)) < maxSubscriptionSize {
					links = f.Parse(decodedStr)
				}
			}
		}
	}

	return links, nil
}

func (f *Fetcher) Parse(content string) []string {
	var links []string
	seen := make(map[string]bool)
	lines := strings.Split(content, "\n")
	
	// Improved URL validation regex
	urlPattern := regexp.MustCompile(`^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://[^\s]+`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || len(line) > maxLineLength {
			continue
		}
		
		// Validate URL format more strictly
		if urlPattern.MatchString(line) {
			// Basic deduplication
			if !seen[line] {
				links = append(links, line)
				seen[line] = true
				
				// Limit to prevent memory exhaustion
				if len(links) >= maxNodesPerSubscription {
					break
				}
			}
		}
	}
	
	return links
}

func (f *Fetcher) SaveCache(data *CacheData) error {
	lastIdx := strings.LastIndex(f.CachePath, "/")
	if lastIdx != -1 {
		dir := f.CachePath[:lastIdx]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(f.CachePath, file, 0600)
}

func (f *Fetcher) LoadCache() (*CacheData, error) {
	file, err := os.ReadFile(f.CachePath)
	if err != nil {
		return nil, err
	}

	var data CacheData
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (f *Fetcher) fetchWithClient(targetURL string, proxyPort int, timeout time.Duration) ([]byte, int, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	if proxyPort > 0 {
		proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, 0, err
	}

	// Set browser-like User-Agent to avoid blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, maxSubscriptionSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
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
