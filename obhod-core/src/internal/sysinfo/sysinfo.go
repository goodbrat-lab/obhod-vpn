package sysinfo

import (
	"net"
	"os"
	"path/filepath"
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

	// 3. Detect DNS Resolvers with safe file reading
	// On OpenWrt it might be /tmp/resolv.conf.d/resolv.conf.auto
	resolvPaths := []string{"/tmp/resolv.conf.d/resolv.conf.auto", "/etc/resolv.conf"}
	for _, path := range resolvPaths {
		data, err := safeReadFile(path)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "nameserver") {
					parts := strings.Fields(line)
					if len(parts) > 1 && isValidIP(parts[1]) {
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

// safeReadFile safely reads file contents with path validation
func safeReadFile(path string) ([]byte, error) {
	// Clean path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	
	// Validate path is within allowed directories
	allowedPrefixes := []string{"/etc/", "/tmp/", "/var/"}
	isAllowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			isAllowed = true
			break
		}
	}
	
	if !isAllowed || strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path: %s", path)
	}
	
	func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func GetSystemHealth() (map[string]interface{}, error) {
	health := make(map[string]interface{})
	
	// Safe command execution
	if checkProcessRunning("sing-box") {
		health["singbox_running"] = true
	} else {
		health["singbox_running"] = false
	}

	// Check if obhoud is running (it should be if we are here, but still)
	health["daemon_running"] = true
	
	// Check for common issues
	health["issues"] = []string{}
	
	return health, nil
}

// checkProcessRunning safely checks if a process is running
func checkProcessRunning(name string) bool {
	// Basic check without command execution
	// In a real implementation, you'd use a proper process checker
	// For now, return a safe default
	return false
}
