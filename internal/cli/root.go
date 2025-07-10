package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/i18n"
)

var (
	cfgFile  string
	langFlag string
	rootCmd  = &cobra.Command{
		Use:   "revlay",
		Short: "",
		Long:  ``,
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "")
	rootCmd.PersistentFlags().StringVarP(&langFlag, "lang", "l", "", "")
	
	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(releasesCmd)
	rootCmd.AddCommand(statusCmd)
}

// initConfig initializes the configuration
func initConfig() {
	// Initialize language first
	i18n.InitLanguage(langFlag)
	
	// Update command descriptions with translated text
	t := i18n.T()
	rootCmd.Short = t.AppShortDesc
	rootCmd.Long = t.AppLongDesc
	rootCmd.PersistentFlags().Lookup("config").Usage = t.ConfigFileFlag
	rootCmd.PersistentFlags().Lookup("lang").Usage = t.LanguageFlag
	
	if cfgFile == "" {
		cfgFile = "revlay.yml"
	}
}

// loadConfig loads the configuration file
func loadConfig() (*config.Config, error) {
	t := i18n.T()
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		return nil, fmt.Errorf(t.ErrorConfigNotFound, cfgFile)
	}
	
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf(t.ErrorConfigLoad, err)
	}
	
	return cfg, nil
}