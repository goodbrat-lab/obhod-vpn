package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/obhod/obhoud/internal/api"
	"github.com/obhod/obhoud/internal/config"
	"github.com/obhod/obhoud/internal/network"
	"github.com/obhod/obhoud/internal/service"
	"github.com/obhod/obhoud/internal/uci"
)

func main() {
	log.Println("Starting obhoud daemon...")

	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Load UCI configuration
	log.Println("Loading configuration...")
	cfg, err := uci.LoadConfig()
	if err != nil {
		log.Printf("Failed to load UCI config, using defaults: %v\n", err)
		// Fallback to empty config for testing if uci is not present
		cfg = &uci.UciConfig{
			Settings: uci.GlobalSettings{
				Enabled:    true,
				Fwmark:     255,
				DnsPort:    15353,
				TproxyPort: 11080,
				LogLevel:   "info",
			},
		}
	}

	sbJson, err := config.GenerateSingBoxConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to generate sing-box config: %v", err)
	}
	log.Printf("Generated sing-box config:\n%s\n", string(sbJson))

	// 2. Setup Network Manager
	log.Println("Initializing network manager...")
	nm := network.NewNetworkManager(cfg.Settings.Fwmark, cfg.Settings.TproxyPort)

	// 3. Setup and Start Service Manager
	log.Println("Initializing service manager...")
	mgr := service.NewManager(cfg, sbJson)
	mgr.ApplyNetworkRules = nm.ApplyRules

	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("Failed to start service manager: %v", err)
	}

	// 4. Start API Server
	log.Println("Starting API server...")
	apiServer := api.NewServer(mgr)
	if err := apiServer.Start(); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}

	// Block until we receive a signal
	<-ctx.Done()
	log.Println("Shutting down obhoud daemon...")

	// Cleanup rules and process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	apiServer.Stop(shutdownCtx)
	nm.Flush()

	log.Println("obhoud stopped gracefully")
}
