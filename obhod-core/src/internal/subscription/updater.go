package subscription

import (
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
	"os/exec"
	"strings"
	"time"
)

func UpdateManual(cachePath string) error {
	fetcher := NewFetcher(cachePath)
	return performUpdate(fetcher)
}

// BackgroundWorker handles periodic updates
func StartUpdater(cachePath string) {
	fetcher := NewFetcher(cachePath)
	ticker := time.NewTicker(24 * time.Hour) // Default
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
		return err
	}

	cache := &CacheData{
		LastUpdate: time.Now(),
		Sections:   make(map[string][]string),
	}

	lines := strings.Split(string(out), "\n")
	urls := make(map[string]string)
	
	for _, line := range lines {
		if strings.Contains(line, ".subscription_url=") {
			parts := strings.SplitN(line, "=", 2)
			keyParts := strings.Split(parts[0], ".")
			sectionName := keyParts[1]
			url := strings.Trim(parts[1], "'")
			urls[sectionName] = url
		}
	}

	if len(urls) == 0 {
		return nil
	}

	for section, url := range urls {
		logger.Info("updater", "fetch", "Updating section %s: %s", section, url)
		links, err := f.Fetch(url)
		if err != nil {
			logger.Error("updater", "fetch", "Failed to fetch %s: %v", section, err)
			continue
		}
		cache.Sections[section] = links
	}

	return f.SaveCache(cache)
}
