package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	resp, err := f.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download subscription: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription fetch returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription body: %v", err)
	}

	content := string(body)
	links := f.Parse(content)

	if len(links) == 0 {
		// Try Base64 decode
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err == nil {
			links = f.Parse(string(decoded))
		}
	}

	return links, nil
}

func (f *Fetcher) Parse(content string) []string {
	var links []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "://") {
			links = append(links, line)
		}
	}
	return links
}

func (f *Fetcher) SaveCache(data *CacheData) error {
	dir := f.CachePath[:strings.LastIndex(f.CachePath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
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
