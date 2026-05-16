package watchdog

import (
	"context"
	"net"
	"os/exec"
	"syscall"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
)

const (
	maxFails        = 3
	singBoxRestarts = 2 // After this many sing-box restarts, escalate to full obhod restart
)

func Start(ctx context.Context, checkInterval time.Duration, mark int) {
	logger.Info("watchdog", "init", "Watchdog started with interval %v, mark %d", checkInterval, mark)
	failCount := 0
	singBoxRestartCount := 0

	// Initial delay to let the system fully come up before we start monitoring
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
			logger.Info("watchdog", "lifecycle", "Watchdog stopping: context cancelled")
			return
		case <-ticker.C:
			if !isWanUp(mark) {
				// WAN is physically down — not our problem, reset counters and wait
				if failCount > 0 {
					logger.Info("watchdog", "connectivity", "WAN is down, resetting DNS fail counter and pausing checks")
					failCount = 0
				}
				continue
			}

			if checkDns() {
				// DNS is healthy
				if failCount > 0 {
					logger.Info("watchdog", "connectivity", "DNS connectivity restored after %d failures", failCount)
				} else {
					logger.Debug("watchdog", "connectivity", "DNS check successful")
				}
				failCount = 0
				singBoxRestartCount = 0
				continue
			}

			// DNS check failed
			failCount++
			logger.Warn("watchdog", "connectivity", "DNS check failed (%d/%d): no response from 127.0.0.1:53", failCount, maxFails)

			if failCount < maxFails {
				continue
			}

			// Threshold reached — take action
			failCount = 0
			singBoxRestartCount++

			if singBoxRestartCount <= singBoxRestarts {
				// First attempts: restart only sing-box (faster, less disruptive)
				logger.Error("watchdog", "recovery", "DNS failure threshold reached. Restarting sing-box (attempt %d/%d)...",
					singBoxRestartCount, singBoxRestarts)
				restartSingBox()
			} else {
				// Repeated failures after sing-box restarts — escalate to full obhod restart
				logger.Error("watchdog", "recovery", "DNS still failing after %d sing-box restarts. Escalating: full obhod restart...",
					singBoxRestartCount-1)
				singBoxRestartCount = 0
				restartObhod()
			}
		}
	}
}

// isWanUp checks physical internet connectivity by dialing a known stable IP.
// It uses SO_MARK to bypass local routing/proxy rules if a mark is provided.
func isWanUp(mark int) bool {
	d := net.Dialer{
		Timeout: 2 * time.Second,
	}

	if mark != 0 {
		d.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, mark)
				if err != nil {
					logger.Warn("watchdog", "wan", "Failed to set SO_MARK %d on socket: %v", mark, err)
				} else {
					logger.Debug("watchdog", "wan", "Successfully set SO_MARK %d for WAN check", mark)
				}
			})
		}
	}

	conn, err := d.Dial("tcp", "1.1.1.1:53")
	if err != nil {
		logger.Debug("watchdog", "wan", "WAN connectivity check failed: %v", err)
		return false
	}
	conn.Close()
	return true
}

// checkDns verifies that the local DNS resolver (sing-box / dnsmasq on :53) is working.
func checkDns() bool {
	r := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			// Query dnsmasq at port 53 — it forwards to sing-box DNS inbound.
			return d.DialContext(ctx, "udp", "127.0.0.1:53")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.LookupHost(ctx, "google.com")
	if err != nil {
		logger.Debug("watchdog", "dns", "DNS lookup failed: %v", err)
	}
	return err == nil
}

// restartSingBox restarts only the sing-box service.
func restartSingBox() {
	logger.Info("watchdog", "action", "Executing: /etc/init.d/sing-box restart")
	cmd := exec.Command("/etc/init.d/sing-box", "restart")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("watchdog", "action", "sing-box restart failed: %v | output: %s", err, string(output))
	} else {
		logger.Info("watchdog", "action", "sing-box restarted successfully")
	}
}

// restartObhod performs a full obhod restart.
func restartObhod() {
	logger.Info("watchdog", "action", "Executing: /usr/bin/obhod restart (full service restart)")
	cmd := exec.Command("/usr/bin/obhod", "restart")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("watchdog", "action", "obhod restart failed: %v | output: %s", err, string(output))
	} else {
		logger.Info("watchdog", "action", "obhod restarted successfully")
	}
}
