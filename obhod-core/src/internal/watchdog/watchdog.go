package watchdog

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/config"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/telegram"
)

const (
	maxFails        = 3
	singBoxRestarts = 2
)

func Start(ctx context.Context, checkInterval time.Duration, mark int) {
	logger.Info("watchdog", "init", "Watchdog started with interval %v, mark %d", checkInterval, mark)
	failCount := 0
	singBoxRestartCount := 0

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
			if !isWanUp(mark) {
				if failCount > 0 {
					logger.Info("watchdog", "connectivity", "WAN is down, pausing checks")
					failCount = 0
				}
				continue
			}

			if checkDns() {
				if failCount > 0 {
					logger.Info("watchdog", "connectivity", "DNS restored")
					if bot != nil {
						bot.SendMessage("✅ <b>DNS Restored</b>. All systems normal.")
					}
				}
				failCount = 0
				singBoxRestartCount = 0
				continue
			}

			failCount++
			logger.Warn("watchdog", "connectivity", "DNS failed (%d/%d)", failCount, maxFails)

			if failCount < maxFails {
				continue
			}

			failCount = 0
			singBoxRestartCount++

			if singBoxRestartCount <= singBoxRestarts {
				logger.Error("watchdog", "recovery", "Restarting sing-box...")
				if bot != nil {
					bot.SendMessage(fmt.Sprintf("⚠️ <b>DNS Failure</b>. Restarting sing-box (attempt %d/%d)...", singBoxRestartCount, singBoxRestarts))
				}
				restartSingBox()
			} else {
				logger.Error("watchdog", "recovery", "Escalating to full restart...")
				if bot != nil {
					bot.SendMessage("🚨 <b>Persistent DNS Failure</b>. Performing full service restart!")
				}
				singBoxRestartCount = 0
				restartObhod()
			}
		}
	}
}

func isWanUp(mark int) bool {
	d := net.Dialer{Timeout: 2 * time.Second}
	if mark != 0 {
		d.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, mark)
			})
		}
	}
	conn, err := d.Dial("tcp", "1.1.1.1:53")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func checkDns() bool {
	r := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", "127.0.0.1:53")
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.LookupHost(ctx, "google.com")
	return err == nil
}

func restartSingBox() {
	exec.Command("/etc/init.d/sing-box", "restart").Run()
}

func restartObhod() {
	exec.Command("/usr/bin/obhod", "restart").Run()
}
