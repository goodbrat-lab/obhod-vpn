package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/obhod/obhoud/internal/dns"
	"github.com/obhod/obhoud/internal/uci"
)

type TunnelStatus struct {
	ID        string
	Status    string // "up" or "down"
	LatencyMs int
}

type Manager struct {
	ConfigPath string
	ConfigData []byte
	DnsAddr    string

	// Hooks for testing
	CheckWANFunc      func(ctx context.Context) error
	RestartSingBoxCmd func(ctx context.Context) error
	LookupDNSFunc     func(ctx context.Context, addr string) error
	ReloadDnsmasqCmd  func() error
	ApplyNetworkRules func() error
	DialTimeoutFunc   func(network, addr string, timeout time.Duration) (net.Conn, error)
	CheckWGFunc       func(ctx context.Context, id, server string) (bool, int)

	// State
	mu            sync.RWMutex
	startTime     time.Time
	wanReady      bool
	dnsHealthy    bool
	tunnelsStatus map[string]TunnelStatus
	tunnelsMu     sync.RWMutex
	UciConfig     *uci.UciConfig
}

func NewManager(cfg *uci.UciConfig, configData []byte) *Manager {
	return &Manager{
		ConfigPath:    "/var/run/obhod/sing-box.json",
		ConfigData:    configData,
		DnsAddr:       fmt.Sprintf("127.0.0.1:%d", cfg.Settings.DnsPort),
		UciConfig:     cfg,
		tunnelsStatus: make(map[string]TunnelStatus),
		CheckWANFunc: func(ctx context.Context) error {
			// Basic ping to 1.1.1.1. In real OpenWrt, ping usually exists.
			cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", "1.1.1.1")
			return cmd.Run()
		},
		RestartSingBoxCmd: func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "/etc/init.d/sing-box", "restart")
			return cmd.Run()
		},
		LookupDNSFunc: func(ctx context.Context, addr string) error {
			// A simple DNS lookup. In a real scenario we'd use a custom resolver or external bin
			// because standard library doesn't easily let us target a specific server IP/port directly
			// without custom Dialer, but let's use a quick nslookup for OpenWrt compatible check
			cmd := exec.CommandContext(ctx, "nslookup", "openwrt.org", "127.0.0.1")
			return cmd.Run()
		},
		ReloadDnsmasqCmd: func() error {
			if err := dns.ConfigureDnsmasq(cfg, cfg.Settings.DnsPort); err != nil {
				return err
			}
			return dns.ReloadDnsmasq()
		},
		ApplyNetworkRules: func() error {
			return nil
		},
		DialTimeoutFunc: net.DialTimeout,
		CheckWGFunc: func(ctx context.Context, id, server string) (bool, int) {
			// Check handshake
			cmd := exec.CommandContext(ctx, "wg", "show", id, "latest-handshakes")
			out, err := cmd.Output()
			if err != nil || len(strings.TrimSpace(string(out))) == 0 {
				return false, 0
			}

			// Parse output: it returns "public_key timestamp"
			parts := strings.Fields(string(out))
			if len(parts) < 2 {
				return false, 0
			}

			ts, _ := strconv.ParseInt(parts[1], 10, 64)
			if ts == 0 {
				return false, 0
			}

			// Handshake exists, now check latency via ping to server
			start := time.Now()
			pingCmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", server)
			if err := pingCmd.Run(); err != nil {
				return true, 0 // Up, but ping failed
			}
			return true, int(time.Since(start).Milliseconds())
		},
	}
}

func (m *Manager) SaveConfig() error {
	dir := filepath.Dir(m.ConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	if err := os.WriteFile(m.ConfigPath, m.ConfigData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	log.Println("Saving sing-box config...")
	if err := m.SaveConfig(); err != nil {
		return err
	}

	log.Println("Waiting for WAN interface...")
	if err := m.waitForWAN(ctx); err != nil {
		return fmt.Errorf("WAN not ready: %w", err)
	}

	m.mu.Lock()
	m.wanReady = true
	m.startTime = time.Now()
	m.mu.Unlock()

	log.Println("Applying network routing and nftables rules...")
	if err := m.ApplyNetworkRules(); err != nil {
		return fmt.Errorf("failed to apply network rules: %w", err)
	}

	log.Println("Reloading dnsmasq cache...")
	if err := m.ReloadDnsmasqCmd(); err != nil {
		log.Printf("Warning: failed to reload dnsmasq: %v\n", err)
	}

	log.Println("Starting sing-box...")
	if err := m.RestartSingBoxCmd(ctx); err != nil {
		return fmt.Errorf("failed to start sing-box: %w", err)
	}

	go m.startDNSMonitor(ctx)
	go m.startTunnelMonitor(ctx)

	return nil
}

func (m *Manager) startTunnelMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Initial check
	m.checkTunnels(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Tunnel monitor stopped")
			return
		case <-ticker.C:
			m.checkTunnels(ctx)
		}
	}
}

func (m *Manager) checkTunnels(ctx context.Context) {
	if m.UciConfig == nil {
		return
	}

	for _, t := range m.UciConfig.Tunnels {
		var status string
		var latency int

		if t.Type == "wireguard" {
			up, lat := m.CheckWGFunc(ctx, t.ID, t.Server)
			if up {
				status = "up"
				latency = lat
			} else {
				status = "down"
			}
		} else {
			// VLESS, etc.
			start := time.Now()
			addr := net.JoinHostPort(t.Server, strconv.Itoa(t.Port))
			conn, err := m.DialTimeoutFunc("tcp", addr, 2*time.Second)
			if err == nil {
				if conn != nil {
					conn.Close()
				}
				status = "up"
				latency = int(time.Since(start).Milliseconds())
			} else {
				status = "down"
			}
		}

		m.tunnelsMu.Lock()
		m.tunnelsStatus[t.ID] = TunnelStatus{
			ID:        t.ID,
			Status:    status,
			LatencyMs: latency,
		}
		m.tunnelsMu.Unlock()
	}
}

func (m *Manager) GetTunnelsStatus() []TunnelStatus {
	m.tunnelsMu.RLock()
	defer m.tunnelsMu.RUnlock()

	var result []TunnelStatus
	// If UciConfig is available, ensure all tunnels are represented
	if m.UciConfig != nil {
		for _, t := range m.UciConfig.Tunnels {
			if s, ok := m.tunnelsStatus[t.ID]; ok {
				result = append(result, s)
			} else {
				result = append(result, TunnelStatus{
					ID:     t.ID,
					Status: "down",
				})
			}
		}
	} else {
		for _, s := range m.tunnelsStatus {
			result = append(result, s)
		}
	}
	return result
}

func (m *Manager) waitForWAN(ctx context.Context) error {
	delay := 2 * time.Second
	maxDelay := 30 * time.Second
	timeout := time.After(2 * time.Minute)

	for {
		err := m.CheckWANFunc(ctx)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for WAN after 2 minutes")
		case <-time.After(delay):
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

func (m *Manager) startDNSMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	failures := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("DNS monitor stopped")
			return
		case <-ticker.C:
			// Run DNS test with short timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := m.LookupDNSFunc(timeoutCtx, m.DnsAddr)
			cancel()

			if err != nil {
				failures++
				m.mu.Lock()
				m.dnsHealthy = false
				m.mu.Unlock()
				log.Printf("DNS check failed (%d/3): %v\n", failures, err)
				if failures >= 3 {
					log.Println("DNS monitor: 3 consecutive failures, restarting sing-box...")
					if restartErr := m.RestartSingBoxCmd(ctx); restartErr != nil {
						log.Printf("Failed to restart sing-box: %v\n", restartErr)
					}
					// Reset failures after restart, delay next check slightly to allow startup
					failures = 0
					time.Sleep(5 * time.Second)
				}
			} else {
				if failures > 0 || !m.IsDNSHealthy() {
					log.Println("DNS check recovered / healthy")
					m.mu.Lock()
					m.dnsHealthy = true
					m.mu.Unlock()
				}
				failures = 0
			}
		}
	}
}

func (m *Manager) IsDNSHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dnsHealthy
}

func (m *Manager) IsWANReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.wanReady
}

func (m *Manager) UptimeSeconds() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.startTime.IsZero() {
		return 0
	}
	return int(time.Since(m.startTime).Seconds())
}

func (m *Manager) RestartSingBox(ctx context.Context) error {
	return m.RestartSingBoxCmd(ctx)
}

