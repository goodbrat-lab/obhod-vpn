package config

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/subscription"
)

func UpdateManual(cachePath string) error {
	uci, err := LoadUCI()
	if err != nil {
		return err
	}
	return UpdateSubscriptions(uci, cachePath)
}

func StartUpdater(ctx context.Context, cachePath string, updateInterval string) {
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
		
		uci, err := LoadUCI()
		if err != nil {
			logger.Error("updater", "loop", "Failed to load UCI config: %v", err)
		} else {
			err = UpdateSubscriptions(uci, cachePath)
			if err != nil {
				logger.Error("updater", "loop", "Update failed: %v", err)
			} else {
				logger.Info("updater", "loop", "Update completed successfully")
				// Trigger config regeneration and reload
				exec.Command("/usr/bin/obhod", "reload").Run()
			}
		}

		select {
		case <-ctx.Done():
			logger.Info("updater", "lifecycle", "Updater stopped")
			return
		case <-ticker.C:
		}
	}
}

func UpdateSubscriptions(uci *UCIConfig, cachePath string) error {
	fetcher := subscription.NewFetcher(cachePath)
	
	// Determine if proxy should be used
	fetcher.ProxyPort = 0
	if uci.Settings.DownloadListsViaProxy {
		if uci.Settings.DownloadListsViaProxySection != "" {
			if sec, ok := uci.Sections[uci.Settings.DownloadListsViaProxySection]; ok && sec.MixedProxyEnabled && sec.MixedProxyPort > 0 {
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
		if fetcher.ProxyPort == 0 {
			fetcher.ProxyPort = 4534
		}
	}

	cache := &subscription.CacheData{
		LastUpdate: time.Now(),
		Sections:   make(map[string][]string),
	}

	successCount := 0
	for name, sec := range uci.Sections {
		if sec.SubscriptionURL != "" {
			logger.Info("updater", "fetch", "Updating section %s: %s", name, subscription.SanitizeURLForLog(sec.SubscriptionURL))
			links, err := fetcher.Fetch(sec.SubscriptionURL)
			if err != nil {
				logger.Error("updater", "fetch", "Failed to fetch %s: %v", name, err)
				continue
			}
			cache.Sections[name] = links
			successCount++
		}
	}

	if successCount > 0 {
		return fetcher.SaveCache(cache)
	}
	return fmt.Errorf("no sections updated successfully")
}
