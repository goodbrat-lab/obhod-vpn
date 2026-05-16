package subscription

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Fetcher struct {
	Client *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
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
		// Basic check for proxy schemes
		if strings.Contains(line, "://") {
			links = append(links, line)
		}
	}
	return links
}
