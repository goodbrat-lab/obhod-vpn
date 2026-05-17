package config

import (
	"sync"
	"time"
)

// AtomicConfig provides atomic configuration operations
type AtomicConfig struct {
	config *UCIConfig
	mutex  sync.RWMutex
}

func NewAtomicConfig() *AtomicConfig {
	return &AtomicConfig{}
}

func (ac *AtomicConfig) Load() (*UCIConfig, error) {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	
	if ac.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	return ac.config, nil
}

func (ac *AtomicConfig) Update(newConfig *UCIConfig) error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	
	// Validate new config before applying
	if err := validateConfig(newConfig); err != nil {
		return fmt.Errorf("config validation failed: %v", err)
	}
	
	// Backup current config
	backup := ac.config
	
	// Apply new config
	ac.config = newConfig
	
	// Verify config is valid
	if err := verifyConfigWorks(newConfig); err != nil {
		// Rollback on failure
		ac.config = backup
		return fmt.Errorf("config verification failed, rolled back: %v", err)
	}
	
	return nil
}

// validateConfig checks configuration validity
func validateConfig(config *UCIConfig) error {
	if config == nil {
		return fmt.Errorf("nil config")
	}
	
	// Validate DNS settings
	if config.Settings.DNSServer == "" {
		return fmt.Errorf("DNS server not specified")
	}
	
	// Validate proxy strings
	for _, section := range config.Sections {
		if section.ProxyString != "" && !isValidProxyString(section.ProxyString) {
			return fmt.Errorf("invalid proxy string in section %s", section.Name)
		}
	}
	
	return nil
}

// verifyConfigWorks checks if configuration works in practice
func verifyConfigWorks(config *UCIConfig) error {
	// Try to generate sing-box config
	_, err := Generate(config)
	if err != nil {
		return fmt.Errorf("failed to generate sing-box config: %v", err)
	}
	
	return nil
}