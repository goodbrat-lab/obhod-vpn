package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"regexp"

	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/config"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/sysinfo"
	"github.com/goodbrat-lab/obhod-vpn/obhod/internal/watchdog"
	"encoding/json"
)

var version = "1.1.39"

func main() {
	watchdogCmd := flag.NewFlagSet("watchdog", flag.ExitOnError)
	interval := watchdogCmd.Duration("interval", 30*time.Second, "Check interval")
	mark := watchdogCmd.Int("mark", 0, "Socket mark (fwmark) for WAN checks (e.g. 2097152 for 0x00200000)")
	
	genConfigCmd := flag.NewFlagSet("generate-config", flag.ExitOnError)
	outputFile := genConfigCmd.String("o", "", "Output file path (default: stdout)")

	updateSubCmd := flag.NewFlagSet("update-subscriptions", flag.ExitOnError)
	cacheFile := updateSubCmd.String("c", "/tmp/obhod/subscriptions.json", "Cache file path")

	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	
	showVersion := flag.Bool("v", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("obhod version %s\n", version)
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		forwardToBackend()
	}

	if flag.Arg(0) == "watchdog" {
		lockFile, err := os.OpenFile("/var/run/obhod_watchdog.lock", os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			err = lockInstance(lockFile)
			if err != nil {
				fmt.Println("Another instance of obhod watchdog is already running.")
				os.Exit(1)
			}
		}
	}

	err := logger.Init(*logLevel)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch flag.Arg(0) {
	case "watchdog":
		watchdogCmd.Parse(flag.Args()[1:])
		// Start subscription updater in background
		updateInterval := "1d"
		if uci, err := config.LoadUCI(); err == nil {
			updateInterval = uci.Settings.UpdateInterval
		}
		go config.StartUpdater(ctx, "", updateInterval)
		watchdog.Start(ctx, *interval, *mark)
	case "auto-setup":
		info, err := sysinfo.GetNetworkInfo()
		if err != nil {
			logger.Error("sysinfo", "main", "Failed to detect network: %v", err)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
	case "health":
		health, err := sysinfo.GetSystemHealth()
		if err != nil {
			logger.Error("sysinfo", "main", "Failed to check health: %v", err)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(health, "", "  ")
		fmt.Println(string(data))
	case "update-subscriptions":
		updateSubCmd.Parse(flag.Args()[1:])
		err := config.UpdateManual(*cacheFile)
		if err != nil {
			logger.Error("config", "main", "Failed to update subscriptions: %v", err)
			os.Exit(1)
		}
	case "generate-config":
		genConfigCmd.Parse(flag.Args()[1:])
		uci, err := config.LoadUCI()
		if err != nil {
			logger.Error("config", "main", "Failed to load UCI: %v", err)
			os.Exit(1)
		}
		sbConfig, err := config.Generate(uci)
		if err != nil {
			logger.Error("config", "main", "Failed to generate config: %v", err)
			os.Exit(1)
		}
		data, err := json.MarshalIndent(sbConfig, "", "  ")
		if err != nil {
			logger.Error("config", "main", "Failed to marshal config: %v", err)
			os.Exit(1)
		}
		if *outputFile != "" {
			err = os.WriteFile(*outputFile, data, 0600)
			if err != nil {
				logger.Error("config", "main", "Failed to write config to %s: %v", *outputFile, err)
				os.Exit(1)
			}
		} else {
			fmt.Println(string(data))
		}
	default:
		forwardToBackend()
	}
}

func forwardToBackend() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmdName := os.Args[1]
	allowedCmds := map[string]bool{
		"start": true, "stop": true, "restart": true, "reload": true,
		"validate": true, "main": true, "start_main": true, "stop_main": true,
		"list_update": true, "backup": true, "restore": true, "check_proxy": true,
		"check_nft": true, "check_nft_rules": true, "check_sing_box": true,
		"check_logs": true, "check_sing_box_logs": true, "check_fakeip": true,
		"clash_api": true, "show_config": true, "show_version": true,
		"show_sing_box_config": true, "show_sing_box_version": true,
		"show_system_info": true, "get_status": true, "get_sing_box_status": true,
		"get_system_info": true, "health": true, "auto_setup": true,
		"check_dns_available": true, "global_check": true,
	}
	if !allowedCmds[cmdName] {
		fmt.Printf("Error: Unknown or restricted command: %s\n\n", cmdName)
		printUsage()
		os.Exit(1)
	}

	safeArgRegex := regexp.MustCompile(`^[^$;|&<>*?\x60\\"'` + "`" + `\x00-\x1F\x7F-\x9F\x0a\x0d]*$`)
	for i, arg := range os.Args[2:] {
		if len(arg) > 255 || !safeArgRegex.MatchString(arg) {
			fmt.Fprintf(os.Stderr, "Error: invalid character or too long argument in parameter %d: %q\n", i+2, arg)
			os.Exit(1)
		}
	}

	backendPath := "/usr/lib/obhod/obhod-backend.sh"
	if _, err := os.Stat(backendPath); os.IsNotExist(err) {
		fmt.Printf("Backend script not found at %s\n", backendPath)
		os.Exit(1)
	}
	
	cmd := exec.Command(backendPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Printf("Failed to run backend script: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func printUsage() {
	fmt.Println("Usage: obhod <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  watchdog             Start connectivity monitoring and scheduled updates")
	// Using dash in commands for consistency
	fmt.Println("  auto-setup           Detect network settings and suggest configuration")
	fmt.Println("  health               Check system health and status")
	fmt.Println("  update-subscriptions Update all proxy subscriptions manually")
	fmt.Println("  generate-config      Generate sing-box configuration from UCI")
	fmt.Println("\nOptions for watchdog:")
	fmt.Println("  -interval duration    Check interval (default 30s)")
	fmt.Println("  -mark int             Socket mark (fwmark) for direct WAN checks")
	fmt.Println("\nOptions for generate-config:")
	fmt.Println("  -o string             Output file path")
}
