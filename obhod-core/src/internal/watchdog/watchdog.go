package watchdog

import (
	"context"
	"net"
	"os/exec"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
)

const (
	maxFails        = 3
	singBoxRestarts = 2 // After this many sing-box restarts, escalate to full obhod restart
)

func Start(ctx context.Context, checkInterval time.Duration) {
	logger.Info("Watchdog started with interval %v", checkInterval)
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
			logger.Info("Watchdog stopping: context cancelled")
			return
		case <-ticker.C:
			if !isWanUp() {
				// WAN is physically down — not our problem, reset counters and wait
				if failCount > 0 {
					logger.Info("WAN is down, resetting DNS fail counter and pausing checks")
					failCount = 0
				}
				continue
			}

			if checkDns() {
				// DNS is healthy
				if failCount > 0 {
					logger.Info("DNS connectivity restored after %d failures", failCount)
				}
				failCount = 0
				singBoxRestartCount = 0
				continue
			}

			// DNS check failed
			failCount++
			logger.Error("DNS check failed (%d/%d): no response from 127.0.0.1:53", failCount, maxFails)

			if failCount < maxFails {
				continue
			}

			// Threshold reached — take action
			failCount = 0
			singBoxRestartCount++

			if singBoxRestartCount <= singBoxRestarts {
				// First attempts: restart only sing-box (faster, less disruptive)
				logger.Error("DNS failure threshold reached. Restarting sing-box (attempt %d/%d)...",
					singBoxRestartCount, singBoxRestarts)
				restartSingBox()
			} else {
				// Repeated failures after sing-box restarts — escalate to full obhod restart
				logger.Error("DNS still failing after %d sing-box restarts. Escalating: full obhod restart...",
					singBoxRestartCount-1)
				singBoxRestartCount = 0
				restartObhod()
			}
		}
	}
}

// isWanUp checks physical internet connectivity by dialing a known stable IP.
func isWanUp() bool {
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", "1.1.1.1:53")
	if err != nil {
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
	return err == nil
}

// restartSingBox restarts only the sing-box service.
// This is the primary recovery action — fast and targeted.
func restartSingBox() {
	logger.Info("Executing: /etc/init.d/sing-box restart")
	cmd := exec.Command("/etc/init.d/sing-box", "restart")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("sing-box restart failed: %v | output: %s", err, string(output))
	} else {
		logger.Info("sing-box restarted successfully")
	}
}

// restartObhod performs a full obhod restart (nft + sing-box + dnsmasq reconfiguration).
// Used as escalation when sing-box restarts alone do not fix the DNS issue.
func restartObhod() {
	logger.Info("Executing: /usr/bin/obhod restart (full service restart)")
	cmd := exec.Command("/usr/bin/obhod", "restart")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("obhod restart failed: %v | output: %s", err, string(output))
	} else {
		logger.Info("obhod restarted successfully")
	}
}
