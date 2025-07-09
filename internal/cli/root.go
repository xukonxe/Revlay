package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "revlay",
		Short: "A modern, fast, dependency-free deployment tool",
		Long: `Revlay is a modern deployment tool that provides atomic deployments,
zero-downtime deployments, and easy rollbacks for traditional server deployments.

It uses a structured directory layout with releases, shared files, and atomic
symlink switching to ensure reliable deployments.`,
		Version: "1.0.0",
	}
)

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is revlay.yml)")
	
	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(releasesCmd)
	rootCmd.AddCommand(statusCmd)
}

// initConfig initializes the configuration
func initConfig() {
	if cfgFile == "" {
		cfgFile = "revlay.yml"
	}
}

// loadConfig loads the configuration file
func loadConfig() (*config.Config, error) {
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file %s not found, run 'revlay init' first", cfgFile)
	}
	
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	return cfg, nil
}