package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/config"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/subscription"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/sysinfo"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/watchdog"
	"encoding/json"
)

var version = "0.4.11"

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
		fmt.Printf("obhoud version %s\n", version)
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	err := logger.Init(*logLevel)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "watchdog":
		watchdogCmd.Parse(os.Args[2:])
		// Start subscription updater in background
		go subscription.StartUpdater("")
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
		updateSubCmd.Parse(os.Args[2:])
		err := subscription.UpdateManual(*cacheFile)
		if err != nil {
			logger.Error("config", "main", "Failed to update subscriptions: %v", err)
			os.Exit(1)
		}
	case "generate-config":
		genConfigCmd.Parse(os.Args[2:])
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
			err = os.WriteFile(*outputFile, data, 0644)
			if err != nil {
				logger.Error("config", "main", "Failed to write config to %s: %v", *outputFile, err)
				os.Exit(1)
			}
		} else {
			fmt.Println(string(data))
		}
	default:
		logger.Error("init", "main", "Unknown command: %s", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: obhoud <command> [options]")
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
