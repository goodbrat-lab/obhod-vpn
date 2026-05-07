package watchdog

import (
	"context"
	"net"
	"os/exec"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
)

func Start(ctx context.Context, checkInterval time.Duration) {
	logger.Info("Watchdog started with interval %v", checkInterval)
	failCount := 0
	maxFails := 3

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if isWanUp() {
				if !checkDns() {
					failCount++
					logger.Error("DNS check failed (%d/%d)", failCount, maxFails)
				} else {
					if failCount > 0 {
						logger.Info("DNS connectivity restored")
					}
					failCount = 0
				}

				if failCount >= maxFails {
					logger.Error("3 consecutive DNS failures detected. Restarting obhod service...")
					restartObhod()
					failCount = 0
				}
			} else {
				// WAN is down, just wait
				if failCount > 0 {
					logger.Info("WAN is down, pausing DNS checks")
					failCount = 0
				}
			}
		}
	}
}

func isWanUp() bool {
	// Simple check by connecting to a stable IP
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", "1.1.1.1:53")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func checkDns() bool {
	// Try to resolve a domain using the default system resolver (which should be pointed to sing-box or dnsmasq)
	// For better precision, we could use a custom resolver pointing to 127.0.0.1:53 (or whatever sing-box uses)
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

func restartObhod() {
	cmd := exec.Command("/etc/init.d/obhod", "restart")
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to restart obhod: %v", err)
	} else {
		logger.Info("Obhod service restarted successfully")
	}
}
