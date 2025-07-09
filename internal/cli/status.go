package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/ssh"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status",
	Long: `Show the current deployment status.
	
This command displays information about the current release,
deployment configuration, and server connection status.`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("📋 Deployment Status\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Application: %s\n", cfg.App.Name)
	fmt.Printf("Server: %s@%s:%d\n", cfg.Server.User, cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Deploy Path: %s\n", cfg.Deploy.Path)
	fmt.Printf("Keep Releases: %d\n", cfg.App.KeepReleases)
	fmt.Printf("───────────────────────────────────────────────────────────────\n")

	// Create SSH client
	sshConfig := &ssh.Config{
		Host:     cfg.Server.Host,
		User:     cfg.Server.User,
		Port:     cfg.Server.Port,
		Password: cfg.Server.Password,
		KeyFile:  cfg.Server.KeyFile,
	}

	client, err := ssh.NewClient(sshConfig)
	if err != nil {
		fmt.Printf("❌ SSH Connection: FAILED (%v)\n", err)
		return nil
	}
	defer client.Close()

	// Test connection
	if err := client.TestConnection(); err != nil {
		fmt.Printf("❌ SSH Connection: FAILED (%v)\n", err)
		return nil
	}
	fmt.Printf("✓ SSH Connection: OK\n")

	// Create deployer
	deployer := deployment.NewDeployer(cfg, client)

	// Check if deployment directories exist
	if exists, err := client.DirExists(cfg.Deploy.Path); err != nil {
		fmt.Printf("❌ Deploy Directory: ERROR (%v)\n", err)
	} else if !exists {
		fmt.Printf("⚠️  Deploy Directory: NOT INITIALIZED\n")
		fmt.Printf("   Run 'revlay deploy' to initialize the deployment structure\n")
		return nil
	} else {
		fmt.Printf("✓ Deploy Directory: OK\n")
	}

	// Get current release
	currentRelease, err := deployer.GetCurrentRelease()
	if err != nil {
		fmt.Printf("❌ Current Release: ERROR (%v)\n", err)
	} else if currentRelease == "" {
		fmt.Printf("⚠️  Current Release: NONE\n")
	} else {
		fmt.Printf("✓ Current Release: %s\n", currentRelease)
	}

	// List releases
	releases, err := deployer.ListReleases()
	if err != nil {
		fmt.Printf("❌ Releases: ERROR (%v)\n", err)
	} else {
		fmt.Printf("✓ Total Releases: %d\n", len(releases))
		if len(releases) > cfg.App.KeepReleases {
			fmt.Printf("⚠️  Old Releases: %d will be cleaned up on next deployment\n", len(releases)-cfg.App.KeepReleases)
		}
	}

	// Show shared paths
	fmt.Printf("───────────────────────────────────────────────────────────────\n")
	fmt.Printf("Shared Paths:\n")
	for _, path := range cfg.Deploy.SharedPaths {
		fmt.Printf("  - %s\n", path)
	}

	// Show hooks
	if len(cfg.Hooks.PreDeploy) > 0 || len(cfg.Hooks.PostDeploy) > 0 ||
		len(cfg.Hooks.PreRollback) > 0 || len(cfg.Hooks.PostRollback) > 0 {
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		fmt.Printf("Hooks:\n")
		
		if len(cfg.Hooks.PreDeploy) > 0 {
			fmt.Printf("  Pre-deploy:\n")
			for _, hook := range cfg.Hooks.PreDeploy {
				fmt.Printf("    - %s\n", hook)
			}
		}
		
		if len(cfg.Hooks.PostDeploy) > 0 {
			fmt.Printf("  Post-deploy:\n")
			for _, hook := range cfg.Hooks.PostDeploy {
				fmt.Printf("    - %s\n", hook)
			}
		}
		
		if len(cfg.Hooks.PreRollback) > 0 {
			fmt.Printf("  Pre-rollback:\n")
			for _, hook := range cfg.Hooks.PreRollback {
				fmt.Printf("    - %s\n", hook)
			}
		}
		
		if len(cfg.Hooks.PostRollback) > 0 {
			fmt.Printf("  Post-rollback:\n")
			for _, hook := range cfg.Hooks.PostRollback {
				fmt.Printf("    - %s\n", hook)
			}
		}
	}

	return nil
}