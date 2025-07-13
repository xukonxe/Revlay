package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/i18n"
)

// NewInitCommand creates the `revlay init` command.
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [app-name]",
		Short: i18n.T().InitShortDesc,
		Long:  i18n.T().InitLongDesc,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}

	cmd.Flags().StringP("dir", "d", ".", i18n.T().InitDirectoryFlag)
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
		fmt.Print(i18n.T().InitPromptName + ": ")
		_, _ = fmt.Scanln(&cfg.App.Name)
		if cfg.App.Name == "" {
			return fmt.Errorf(i18n.T().InitFailed, "application name cannot be empty")
		}
	}

	// Create project directory inside the base directory
	projectDir := filepath.Join(baseDir, cfg.App.Name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf(i18n.T().InitFailed, fmt.Sprintf("failed to create project directory at %s: %v", projectDir, err))
	}

	configPath := filepath.Join(projectDir, "revlay.yml")

	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf(i18n.T().InitFailed, fmt.Sprintf("configuration file '%s' already exists. Use --force to overwrite", configPath))
	}

	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf(i18n.T().InitFailed, fmt.Sprintf("failed to save configuration: %v", err))
	}

	fmt.Println(color.Green("✔ " + i18n.Sprintf(i18n.T().InitSuccess, configPath)))
	return nil
}
