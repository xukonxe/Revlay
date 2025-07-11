package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xukonxe/revlay/internal/config"
	"gopkg.in/yaml.v3"
)

// loadConfig loads the configuration file.
func loadConfig(cfgFile string) (*config.Config, error) {
	if cfgFile == "" {
		cfgFile = "revlay.yml"
	}

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file '%s' not found", cfgFile)
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set the root path based on the config file's directory
	absPath, err := filepath.Abs(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path for config file: %w", err)
	}
	cfg.RootPath = filepath.Dir(absPath)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
} 