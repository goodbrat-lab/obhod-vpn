package watchdog

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/config"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/telegram"
)

type RestartHistory struct {
	Timestamps    []time.Time `json:"timestamps"`
	LastAlertTime time.Time   `json:"last_alert_time"`
}

func loadRestartHistory() RestartHistory {
	var history RestartHistory
	data, err := os.ReadFile("/tmp/obhod/restart_history.json")
	if err != nil {
		return history
	}
	if err := json.Unmarshal(data, &history); err != nil {
		logger.Warn("watchdog", "telemetry", "Failed to parse restart history JSON, starting fresh: %v", err)
		return RestartHistory{}
	}
	return history
}

func saveRestartHistory(history RestartHistory) {
	_ = os.MkdirAll("/tmp/obhod", 0755)
	data, err := json.Marshal(history)
	if err != nil {
		logger.Error("watchdog", "telemetry", "Failed to marshal restart history: %v", err)
		return
	}
	tmpPath := "/tmp/obhod/restart_history.json.tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		logger.Error("watchdog", "telemetry", "Failed to write temp restart history file: %v", err)
		return
	}
	if err := os.Rename(tmpPath, "/tmp/obhod/restart_history.json"); err != nil {
		logger.Error("watchdog", "telemetry", "Failed to atomically rename restart history file: %v", err)
	}
}

func recordRestart(bot *telegram.Bot) {
	history := loadRestartHistory()
	now := time.Now()
	history.Timestamps = append(history.Timestamps, now)

	// Clean up old timestamps (older than 60 minutes)
	cutoff := now.Add(-60 * time.Minute)
	var activeTimestamps []time.Time
	for _, t := range history.Timestamps {
		if t.After(cutoff) {
			activeTimestamps = append(activeTimestamps, t)
		}
	}
	history.Timestamps = activeTimestamps

	// Check threshold: 5 or more restarts in 60 minutes
	if len(history.Timestamps) >= 5 {
		if now.Sub(history.LastAlertTime) > 60*time.Minute {
			logger.Warn("watchdog", "telemetry", "High restart rate detected: %d restarts in the last 60 minutes", len(history.Timestamps))
			if bot != nil {
				err := bot.SendMessage(fmt.Sprintf("🚨 <b>Channel Instability Alert</b>: sing-box has been restarted %d times in the last 60 minutes. The connection might be unstable.", len(history.Timestamps)))
				if err != nil {
					logger.Error("watchdog", "telemetry", "Failed to send Telegram alert: %v", err)
				}
			}
			history.LastAlertTime = now
		}
	}

	saveRestartHistory(history)
}

const (
	maxFails        = 3
	singBoxRestarts = 2
)

// Global state with mutex protection
var (
 watchdogMutex sync.Mutex
 failCount     int
 singBoxRestartCount int
)

func Start(ctx context.Context, checkInterval time.Duration, mark int) {
	logger.Info("watchdog", "init", "Watchdog started with interval %v, mark %d", checkInterval, mark)
	
	// Reset counters on start
	watchdogMutex.Lock()
	failCount = 0
	singBoxRestartCount = 0
	watchdogMutex.Unlock()

	uci, _ := config.LoadUCI()
	var bot *telegram.Bot
	if uci != nil && uci.Settings.TelegramEnabled {
		bot = telegram.NewBot(uci.Settings.TelegramToken, uci.Settings.TelegramChatID)
		bot.SendMessage("🚀 <b>Obhod Watchdog</b> started on " + uci.Settings.ConfigPath)
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(60 * time.Second):
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("watchdog", "lifecycle", "Watchdog stopping")
			return
		case <-ticker.C:
			processWanCheck(ctx, bot, mark)
		}
	}
}

func processWanCheck(ctx context.Context, bot *telegram.Bot, mark int) {
	watchdogMutex.Lock()
	defer watchdogMutex.Unlock()
	
	wanUp := isWanUp(mark)
	logger.Debug("watchdog", "connectivity", "WAN link check: up=%v", wanUp)
	if !wanUp {
		if failCount > 0 {
			logger.Info("watchdog", "connectivity", "WAN is down, pausing checks")
			failCount = 0
		}
		return
	}

	dnsUp := checkDns()
	logger.Debug("watchdog", "connectivity", "Local DNS via sing-box check: working=%v", dnsUp)
	if dnsUp {
		if failCount > 0 {
			logger.Info("watchdog", "connectivity", "DNS restored")
			if bot != nil {
				bot.SendMessage("✅ <b>DNS Restored</b>. All systems normal.")
			}
		}
		failCount = 0
		singBoxRestartCount = 0
		return
	}

	// Differential DNS Check: verify if DNS resolution works directly via WAN (🛡️ Oshi C)
	var wanDnsWorks bool
	domains := []string{"google.com", "cloudflare.com", "yandex.ru"}
	for _, domain := range domains {
		if checkDnsDirect(domain, mark) {
			wanDnsWorks = true
			break
		}
	}

	if !wanDnsWorks {
		// Both local DNS (sing-box) and direct WAN DNS are failing.
		// This indicates a global WAN DNS or connectivity outage.
		// We do NOT blame sing-box, just warn and wait.
		if failCount > 0 {
			logger.Info("watchdog", "connectivity", "WAN DNS or connectivity is down, pausing recovery checks")
			failCount = 0
		}
		return
	}

	failCount++
	logger.Warn("watchdog", "connectivity", "DNS failed via sing-box, but works directly via WAN (%d/%d)", failCount, maxFails)

	if failCount < maxFails {
		return
	}

	failCount = 0
	singBoxRestartCount++

	if singBoxRestartCount <= singBoxRestarts {
		logger.Error("watchdog", "recovery", "Restarting sing-box...")
		if bot != nil {
			bot.SendMessage(fmt.Sprintf("⚠️ <b>DNS Failure</b>. Restarting sing-box (attempt %d/%d)...", singBoxRestartCount, singBoxRestarts))
		}
		recordRestart(bot)
		restartSingBox()
	} else {
		logger.Error("watchdog", "recovery", "Escalating to full restart...")
		if bot != nil {
			bot.SendMessage("🚨 <b>Persistent DNS Failure</b>. Performing full service restart!")
		}
		singBoxRestartCount = 0
		recordRestart(bot)
		restartObhod()
	}
}

func isWanUp(mark int) bool {
	targets := []struct {
		network string
		address string
	}{
		{"tcp", "77.88.8.8:53"},
		{"tcp", "1.1.1.1:53"},
		{"tcp", "8.8.8.8:53"},
		{"tcp", "1.1.1.1:80"},
		{"tcp", "8.8.8.8:80"},
	}

	for _, target := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err := dialWithMark(ctx, target.network, target.address, mark, 2*time.Second)
		cancel()
		if err == nil {
			conn.Close()
			logger.Debug("watchdog", "connectivity", "WAN connection to %s:%s succeeded", target.network, target.address)
			return true
		}
		logger.Debug("watchdog", "connectivity", "WAN connection to %s:%s failed: %v", target.network, target.address, err)
	}
	return false
}

func checkDns() bool {
	domains := []string{"google.com", "cloudflare.com", "yandex.ru"}
	r := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			// sing-box 1.12+: DNS goes via dnsmasq (127.0.0.1:53) -> nft tproxy -> sing-box hijack-dns
			// Old approach was direct DNS inbound at 127.0.0.42:53, which no longer exists
			return d.DialContext(ctx, "udp", "127.0.0.1:53")
		},
	}

	for _, domain := range domains {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		_, err := r.LookupHost(ctx, domain)
		cancel()
		if err == nil {
			logger.Debug("watchdog", "connectivity", "Local DNS query for %s succeeded", domain)
			return true
		}
		logger.Debug("watchdog", "connectivity", "Local DNS query for %s failed: %v", domain, err)
	}
	return false
}

func checkDnsDirect(domain string, mark int) bool {
	dnsServers := []string{"8.8.8.8:53", "1.1.1.1:53", "77.88.8.8:53"}
	for _, dnsServer := range dnsServers {
		r := net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialWithMark(ctx, "udp", dnsServer, mark, 2*time.Second)
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := r.LookupHost(ctx, domain)
		cancel()
		if err == nil {
			logger.Debug("watchdog", "connectivity", "Direct WAN DNS check via %s succeeded for %s", dnsServer, domain)
			return true
		}
		logger.Debug("watchdog", "connectivity", "Direct WAN DNS check via %s failed for %s: %v", dnsServer, domain, err)
	}
	return false
}

func restartSingBox() {
	exec.Command("/etc/init.d/sing-box", "restart").Run()
}

func restartObhod() {
	exec.Command("/usr/bin/obhod", "restart").Run()
}
