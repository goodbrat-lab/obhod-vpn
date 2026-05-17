package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goodbrat-lab/obhod-vpn/obhod/src/internal/config"
	"github.com/goodbrat-lab/obhod-vpn/obhod/src/internal/logger"
	"github.com/goodbrat-lab/obhod-vpn/obhod/src/internal/subscription"
	"github.com/goodbrat-lab/obhod-vpn/obhod/src/internal/sysinfo"
	"github.com/goodbrat-lab/obhod-vpn/obhod/src/internal/watchdog"
)

const (
	// Safe constants for encrypted storage
	aesKey = "32-byte-long-key-for-obhod-secure" // In production, use proper key management
)

type EncryptedStorage struct {
	Data map[string]string `json:"data"`
}

func main() {
	// Graceful shutdown setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("main", "lifecycle", "Obhod shutting down...")
		cancel()
	}()

	// Initialize logger
	err := logger.Init("info")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("main", "lifecycle", "Obhod starting...")
	
	// Start background tasks
	go startBackgroundServices(ctx)
	
	// Main server loop
	<-ctx.Done()
	logger.Info("main", "lifecycle", "Obhod stopped")
}

func startBackgroundServices(ctx context.Context) {
	// Start watchdog
	go watchdog.Start(ctx, 30*time.Second, 0x00100000)
	
	// Start subscription updater
	go subscription.StartUpdater("/tmp/obhod/subscriptions.json")
	
	// Health check loop
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			performHealthCheck()
		}
	}
}

func performHealthCheck() {
	info, err := sysinfo.GetNetworkInfo()
	if err != nil {
		logger.Error("health", "check", "Failed to get network info: %v", err)
		return
	}

	health, err := sysinfo.GetSystemHealth()
	if err != nil {
		logger.Error("health", "check", "Failed to get system health: %v", err)
		return
	}

	logger.Info("health", "check", "WAN: %s, LocalIP: %s, SingBox: %v", 
		info.WANInterface, info.LocalIP, health["singbox_running"])
}

// encryptSecret encrypts sensitive data using AES-GCM
func encryptSecret(plaintext string) (string, error) {
	if len(aesKey) != 32 {
		return "", fmt.Errorf("invalid key size")
	}

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptSecret decrypts sensitive data using AES-GCM
func decryptSecret(ciphertext string) (string, error) {
	if len(aesKey) != 32 {
		return "", fmt.Errorf("invalid key size")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// saveEncryptedConfig saves configuration with encrypted secrets
func saveEncryptedConfig(config *config.UCIConfig, path string) error {
	storage := EncryptedStorage{
		Data: make(map[string]string),
	}

	// Encrypt sensitive fields
	if config.Settings.TelegramToken != "" {
		encrypted, err := encryptSecret(config.Settings.TelegramToken)
		if err != nil {
			return err
		}
		storage.Data["telegram_token"] = encrypted
	}

	if config.Settings.TelegramChatID != "" {
		encrypted, err := encryptSecret(config.Settings.TelegramChatID)
		if err != nil {
			return err
		}
		storage.Data["telegram_chat_id"] = encrypted
	}

	// Save to file
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}