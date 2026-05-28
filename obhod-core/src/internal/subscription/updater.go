package subscription

import (
	"fmt"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/logger"
	"os/exec"
	"strings"
	"time"
)
func UpdateManual(cachePath string) error {
	fetcher := NewFetcher(cachePath)
	return performUpdate(fetcher)
}

// BackgroundWorker handles periodic updates
func StartUpdater(cachePath string, updateInterval string) {
	fetcher := NewFetcher(cachePath)
	interval := 24 * time.Hour // Default 1d

	switch updateInterval {
	case "1h":
		interval = 1 * time.Hour
	case "3h":
		interval = 3 * time.Hour
	case "12h":
		interval = 12 * time.Hour
	case "1d":
		interval = 24 * time.Hour
	case "3d":
		interval = 3 * 24 * time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		logger.Info("updater", "loop", "Starting scheduled subscription update...")
		
		// 1. Get UCI config to know what to update
		// We'll use a simplified approach: call 'uci show' or implement a real loader
		// For now, let's just trigger a manual update logic here
		err := performUpdate(fetcher)
		if err != nil {
			logger.Error("updater", "loop", "Update failed: %v", err)
		} else {
			logger.Info("updater", "loop", "Update completed successfully")
			// Trigger config regeneration and reload
			exec.Command("/usr/bin/obhod", "reload").Run()
		}

		<-ticker.C
	}
}

func performUpdate(f *Fetcher) error {
	// 1. Load UCI to find all subscription URLs
	cmd := exec.Command("uci", "-q", "show", "obhod")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to load UCI config: %v", err)
	}

	cache := &CacheData{
		LastUpdate: time.Now(),
		Sections:   make(map[string][]string),
	}

	lines := strings.Split(string(out), "\n")
	urls := make(map[string]string)

	var downloadViaProxy bool
	var proxySection string
	mixedPorts := make(map[string]int)
	mixedEnabled := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, ".subscription_url=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				keyParts := strings.Split(parts[0], ".")
				if len(keyParts) >= 3 {
					sectionName := keyParts[1]
					url := strings.Trim(parts[1], "'")
					if len(url) > 0 && len(url) <= 1000 && strings.Contains(url, "://") {
						urls[sectionName] = url
					}
				}
			}
		} else if strings.Contains(line, ".download_lists_via_proxy=") {
			downloadViaProxy = strings.HasSuffix(line, "=1")
		} else if strings.Contains(line, ".download_lists_via_proxy_section=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				proxySection = strings.Trim(parts[1], "'")
			}
		} else if strings.Contains(line, ".mixed_proxy_enabled=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				keyParts := strings.Split(parts[0], ".")
				if len(keyParts) >= 2 {
					secName := keyParts[1]
					mixedEnabled[secName] = strings.HasSuffix(line, "=1")
				}
			}
		} else if strings.Contains(line, ".mixed_proxy_port=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				keyParts := strings.Split(parts[0], ".")
				if len(keyParts) >= 2 {
					secName := keyParts[1]
					var port int
					fmt.Sscanf(strings.Trim(parts[1], "'"), "%d", &port)
					mixedPorts[secName] = port
				}
			}
		}
	}

	// Set ProxyPort in fetcher
	f.ProxyPort = 0
	if downloadViaProxy {
		if proxySection != "" {
			if mixedEnabled[proxySection] && mixedPorts[proxySection] > 0 {
				f.ProxyPort = mixedPorts[proxySection]
			}
		} else {
			for sec, enabled := range mixedEnabled {
				if enabled && mixedPorts[sec] > 0 {
					f.ProxyPort = mixedPorts[sec]
					break
				}
			}
		}
	}

	if len(urls) == 0 {
		return nil
	}

	// Track successful updates
	successCount := 0
	for section, url := range urls {
		logger.Info("updater", "fetch", "Updating section %s: %s", section, url)
		links, err := f.Fetch(url)
		if err != nil {
			logger.Error("updater", "fetch", "Failed to fetch %s: %v", section, err)
			continue
		}
		cache.Sections[section] = links
		successCount++
	}
	
	// Only save update if any section was successfully updated
	if successCount > 0 {
		return f.SaveCache(cache)
	}
	
	return fmt.Errorf("no sections updated successfully")
}
