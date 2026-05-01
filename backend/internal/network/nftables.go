package network

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"text/template"
)

type NetworkManager struct {
	Fwmark     int
	TproxyPort int
	TableID    int

	// Hooks for testing
	ExecCmd func(cmd string, args ...string) error
}

func NewNetworkManager(fwmark, tproxyPort int) *NetworkManager {
	return &NetworkManager{
		Fwmark:     fwmark,
		TproxyPort: tproxyPort,
		TableID:    105, // A dedicated routing table ID for Obhod
		ExecCmd: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			return cmd.Run()
		},
	}
}

const nftRulesTemplate = `
table inet obhod {
	chain mangle {
		type filter hook prerouting priority -150; policy accept;
		ip daddr 198.18.0.0/15 meta l4proto tcp meta mark set {{.Fwmark}} counter
		ip daddr 198.18.0.0/15 meta l4proto udp meta mark set {{.Fwmark}} counter
	}
	chain mangle_output {
		type route hook output priority -150; policy accept;
		meta mark {{.Fwmark}} counter return
		ip daddr 198.18.0.0/15 meta l4proto tcp meta mark set {{.Fwmark}} counter
		ip daddr 198.18.0.0/15 meta l4proto udp meta mark set {{.Fwmark}} counter
	}
	chain proxy {
		type filter hook prerouting priority -100; policy accept;
		meta mark & {{.Fwmark}} == {{.Fwmark}} meta l4proto tcp tproxy ip to 127.0.0.1:{{.TproxyPort}} counter
		meta mark & {{.Fwmark}} == {{.Fwmark}} meta l4proto udp tproxy ip to 127.0.0.1:{{.TproxyPort}} counter
	}
}
`

func (nm *NetworkManager) ApplyRules() error {
	log.Println("Applying network routing and nftables rules...")

	// 1. IP Route
	// ip route add local default dev lo table <TableID>
	_ = nm.ExecCmd("ip", "route", "add", "local", "default", "dev", "lo", "table", fmt.Sprintf("%d", nm.TableID))

	// 2. IP Rule
	// ip rule add fwmark <Fwmark>/<Fwmark> table <TableID>
	_ = nm.ExecCmd("ip", "rule", "add", "fwmark", fmt.Sprintf("%d/%d", nm.Fwmark, nm.Fwmark), "table", fmt.Sprintf("%d", nm.TableID))

	// 3. Nftables
	tmpl, err := template.New("nft").Parse(nftRulesTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse nft template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nm); err != nil {
		return fmt.Errorf("failed to execute nft template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "obhod_nft_*.conf")
	if err != nil {
		return fmt.Errorf("failed to create temp file for nft: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	if err := nm.ExecCmd("nft", "-f", tmpFile.Name()); err != nil {
		return fmt.Errorf("nft command failed: %w", err)
	}

	return nil
}

func (nm *NetworkManager) Flush() {
	log.Println("Flushing network routing and nftables rules...")

	// Ignore errors during flush since items might not exist
	_ = nm.ExecCmd("nft", "delete", "table", "inet", "obhod")
	_ = nm.ExecCmd("ip", "rule", "del", "fwmark", fmt.Sprintf("%d/%d", nm.Fwmark, nm.Fwmark), "table", fmt.Sprintf("%d", nm.TableID))
	_ = nm.ExecCmd("ip", "route", "del", "local", "default", "dev", "lo", "table", fmt.Sprintf("%d", nm.TableID))
}
