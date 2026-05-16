package sysinfo

import (
	"net"
	"os/exec"
	"strings"
)

type NetworkInfo struct {
	WANInterface string   `json:"wan_interface"`
	LocalIP      string   `json:"local_ip"`
	DNSResolvers []string `json:"dns_resolvers"`
	IsIPv6Ready  bool     `json:"is_ipv6_ready"`
}

func GetNetworkInfo() (*NetworkInfo, error) {
	info := &NetworkInfo{
		DNSResolvers: []string{},
	}

	// 1. Detect WAN Interface (via route command)
	cmd := exec.Command("ip", "route", "show", "default")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			parts := strings.Fields(lines[0])
			for i, part := range parts {
				if part == "dev" && i+1 < len(parts) {
					info.WANInterface = parts[i+1]
					break
				}
			}
		}
	}

	// 2. Detect Local IP
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					info.LocalIP = ipnet.IP.String()
					break
				}
			}
		}
	}

	// 3. Detect DNS Resolvers (via resolv.conf)
	// On OpenWrt it might be /tmp/resolv.conf.d/resolv.conf.auto
	resolvPaths := []string{"/tmp/resolv.conf.d/resolv.conf.auto", "/etc/resolv.conf"}
	for _, path := range resolvPaths {
		data, err := exec.Command("cat", path).Output()
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "nameserver") {
					parts := strings.Fields(line)
					if len(parts) > 1 {
						info.DNSResolvers = append(info.DNSResolvers, parts[1])
					}
				}
			}
			if len(info.DNSResolvers) > 0 {
				break
			}
		}
	}

	// 4. IPv6 Check
	cmdV6 := exec.Command("ip", "-6", "route", "show", "default")
	outV6, _ := cmdV6.Output()
	info.IsIPv6Ready = len(strings.TrimSpace(string(outV6))) > 0

	return info, nil
}

func GetSystemHealth() (map[string]interface{}, error) {
	health := make(map[string]interface{})
	
	// Check if sing-box is running
	cmd := exec.Command("pgrep", "-x", "sing-box")
	err := cmd.Run()
	health["singbox_running"] = (err == nil)

	// Check if obhoud is running (it should be if we are here, but still)
	health["daemon_running"] = true
	
	// Check for common issues
	health["issues"] = []string{}
	
	return health, nil
}
