package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhoud/internal/watchdog"
)

var version = "0.3.1"

func main() {
	watchdogCmd := flag.NewFlagSet("watchdog", flag.ExitOnError)
	interval := watchdogCmd.Duration("interval", 30*time.Second, "Check interval")
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
		watchdog.Start(ctx, *interval)
	default:
		logger.Error("init", "main", "Unknown command: %s", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: obhoud <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  watchdog    Start connectivity monitoring")
	fmt.Println("\nOptions for watchdog:")
	fmt.Println("  -interval duration    Check interval (default 30s)")
}
