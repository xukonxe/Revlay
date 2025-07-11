package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/color"
)

// NewInitCommand creates the `revlay init` command.
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [app-name]",
		Short: "Initialize a new revlay project",
		Long:  `Creates a new revlay.yml configuration file in a directory for your application.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}

	cmd.Flags().StringP("dir", "d", ".", "Base directory where the application folder will be created")
	cmd.Flags().BoolP("force", "f", false, "Overwrite existing revlay.yml if it exists")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	baseDir, _ := cmd.Flags().GetString("dir")
	force, _ := cmd.Flags().GetBool("force")

	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	cfg := config.DefaultConfig()

	if appName != "" {
		cfg.App.Name = appName
	}

	if cfg.App.Name == "" {
		fmt.Print("Please enter the name of your application: ")
		_, _ = fmt.Scanln(&cfg.App.Name)
		if cfg.App.Name == "" {
			return fmt.Errorf("application name cannot be empty")
		}
	}

	// Create project directory inside the base directory
	projectDir := filepath.Join(baseDir, cfg.App.Name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory at %s: %w", projectDir, err)
	}

	configPath := filepath.Join(projectDir, "revlay.yml")

	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("configuration file '%s' already exists. Use --force to overwrite", configPath)
	}

	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println(color.Green("✔ Revlay project initialized successfully!"))
	fmt.Printf("Configuration file created at: %s\n", color.Cyan(configPath))
	return nil
}