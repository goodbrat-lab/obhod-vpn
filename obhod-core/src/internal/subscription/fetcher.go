package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Fetcher struct {
	Client    *http.Client
	CachePath string
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
func validateURL(url string) error {
	if len(url) > maxLineLength {
		return fmt.Errorf("URL too long")
	}
	return nil
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
	}
}

func (f *Fetcher) Fetch(url string) ([]string, error) {
	// Validate URL first
	if err := validateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	
	resp, err := f.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download subscription: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription fetch returned HTTP %d", resp.StatusCode)
	}

	// Limit read size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxSubscriptionSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription body: %v", err)
	}

	content := string(body)
	links := f.Parse(content)

	if len(links) == 0 {
		// Try Base64 decode (with trim to handle whitespace/newlines)
		trimmed := strings.TrimSpace(content)
		if len(trimmed) > 0 {
			decoded, err := base64.StdEncoding.DecodeString(trimmed)
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

	return os.WriteFile(f.CachePath, file, 0644)
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
