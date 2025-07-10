package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/i18n"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "",
	Long:  ``,
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
	// Command descriptions will be updated in initConfig
	initCmd.Flags().StringVarP(&initAppName, "name", "n", "", "")
	initCmd.Flags().StringVarP(&initRepository, "repo", "r", "", "Repository URL")
	initCmd.Flags().StringVarP(&initHost, "host", "H", "", "")
	initCmd.Flags().StringVarP(&initUser, "user", "u", "", "")
	initCmd.Flags().StringVarP(&initPath, "path", "p", "", "")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing config file")
	
	// Update command descriptions when config is initialized
	cobra.OnInitialize(func() {
		t := i18n.T()
		initCmd.Short = t.InitShortDesc
		initCmd.Long = t.InitLongDesc
		initCmd.Flags().Lookup("name").Usage = t.InitNameFlag
		initCmd.Flags().Lookup("host").Usage = t.InitHostFlag
		initCmd.Flags().Lookup("user").Usage = t.InitUserFlag
		initCmd.Flags().Lookup("path").Usage = t.InitPathFlag
	})
}

func runInit(cmd *cobra.Command, args []string) error {
	t := i18n.T()
	
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
			return fmt.Errorf(t.InitFailed, err)
		}
	}

	// Save config
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		return fmt.Errorf(t.InitFailed, err)
	}

	fmt.Printf(t.InitSuccess+"\n", cfgFile)
	
	return nil
}

func interactiveConfig(cfg *config.Config) error {
	t := i18n.T()
	
	fmt.Println("Interactive configuration:")
	fmt.Println("Press Enter to use default values shown in [brackets]")
	fmt.Println()

	// Application name
	fmt.Printf("%s [%s]: ", t.InitPromptName, cfg.App.Name)
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
	fmt.Printf("%s [%s]: ", t.InitPromptHost, cfg.Server.Host)
	fmt.Scanln(&input)
	if input != "" {
		cfg.Server.Host = input
	}

	// SSH user
	fmt.Printf("%s [%s]: ", t.InitPromptUser, cfg.Server.User)
	fmt.Scanln(&input)
	if input != "" {
		cfg.Server.User = input
	}

	// Deployment path
	fmt.Printf("%s [%s]: ", t.InitPromptPath, cfg.Deploy.Path)
	fmt.Scanln(&input)
	if input != "" {
		cfg.Deploy.Path = input
	}

	return nil
}