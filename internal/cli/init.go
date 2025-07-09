package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Revlay project",
	Long: `Initialize a new Revlay project by creating a revlay.yml configuration file.
	
This command creates a revlay.yml file in the current directory with default
configuration values that you can customize for your deployment needs.`,
	RunE: runInit,
}

var (
	initAppName    string
	initRepository string
	initHost       string
	initUser       string
	initPath       string
	initForce      bool
)

func init() {
	initCmd.Flags().StringVarP(&initAppName, "name", "n", "", "Application name")
	initCmd.Flags().StringVarP(&initRepository, "repo", "r", "", "Repository URL")
	initCmd.Flags().StringVarP(&initHost, "host", "H", "", "Server hostname")
	initCmd.Flags().StringVarP(&initUser, "user", "u", "", "SSH username")
	initCmd.Flags().StringVarP(&initPath, "path", "p", "", "Deployment path")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config file")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if config file already exists
	if _, err := os.Stat(cfgFile); err == nil && !initForce {
		return fmt.Errorf("config file %s already exists, use --force to overwrite", cfgFile)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Apply command line overrides
	if initAppName != "" {
		cfg.App.Name = initAppName
	}
	if initRepository != "" {
		cfg.App.Repository = initRepository
	}
	if initHost != "" {
		cfg.Server.Host = initHost
	}
	if initUser != "" {
		cfg.Server.User = initUser
	}
	if initPath != "" {
		cfg.Deploy.Path = initPath
	}

	// Interactive configuration if no flags provided
	if initAppName == "" && initRepository == "" && initHost == "" && initUser == "" && initPath == "" {
		if err := interactiveConfig(cfg); err != nil {
			return fmt.Errorf("interactive configuration failed: %w", err)
		}
	}

	// Save config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Configuration file created: %s\n", cfgFile)
	fmt.Println("✓ Edit the configuration file to customize your deployment settings")
	fmt.Println("✓ Run 'revlay deploy' to start your first deployment")

	return nil
}

func interactiveConfig(cfg *config.Config) error {
	fmt.Println("Interactive configuration:")
	fmt.Println("Press Enter to use default values shown in [brackets]")
	fmt.Println()

	// Application name
	fmt.Printf("Application name [%s]: ", cfg.App.Name)
	var input string
	fmt.Scanln(&input)
	if input != "" {
		cfg.App.Name = input
	}

	// Repository
	fmt.Printf("Repository URL [%s]: ", cfg.App.Repository)
	fmt.Scanln(&input)
	if input != "" {
		cfg.App.Repository = input
	}

	// Server host
	fmt.Printf("Server hostname [%s]: ", cfg.Server.Host)
	fmt.Scanln(&input)
	if input != "" {
		cfg.Server.Host = input
	}

	// SSH user
	fmt.Printf("SSH username [%s]: ", cfg.Server.User)
	fmt.Scanln(&input)
	if input != "" {
		cfg.Server.User = input
	}

	// Deployment path
	fmt.Printf("Deployment path [%s]: ", cfg.Deploy.Path)
	fmt.Scanln(&input)
	if input != "" {
		cfg.Deploy.Path = input
	}

	return nil
}