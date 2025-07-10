package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ssh"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	Long:  ``,
	RunE: runStatus,
}

func init() {
	// Update command descriptions when config is initialized
	cobra.OnInitialize(func() {
		t := i18n.T()
		statusCmd.Short = t.StatusShortDesc
		statusCmd.Long = t.StatusLongDesc
	})
}

func runStatus(cmd *cobra.Command, args []string) error {
	t := i18n.T()
	
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("📋 Deployment Status\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("%s: %s\n", t.StatusAppName, cfg.App.Name)
	fmt.Printf("%s: %s@%s:%d\n", t.StatusServerInfo, cfg.Server.User, cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("%s: %s\n", t.StatusDeployPath, cfg.Deploy.Path)
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
		fmt.Printf("❌ %s: ERROR (%v)\n", t.StatusCurrentRelease, err)
	} else if currentRelease == "" {
		fmt.Printf("⚠️  %s: %s\n", t.StatusCurrentRelease, t.StatusNoRelease)
	} else {
		fmt.Printf("✓ %s: %s\n", t.StatusCurrentRelease, currentRelease)
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

	// Show deployment mode
	fmt.Printf("───────────────────────────────────────────────────────────────\n")
	fmt.Printf("%s: %s\n", t.DeploymentMode, cfg.Deploy.Mode)
	
	if cfg.Deploy.Mode == "zero_downtime" {
		fmt.Printf("  - %s: %d\n", t.ServicePort, cfg.Service.Port)
		fmt.Printf("  - Alternative Port: %d\n", cfg.Service.AltPort)
		fmt.Printf("  - %s: %s\n", t.ServiceHealthCheck, cfg.Service.HealthCheck)
	} else {
		fmt.Printf("  - %s: %s\n", t.ServiceCommand, cfg.Service.Command)
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
			fmt.Printf("  %s:\n", t.DryRunPreDeploy)
			for _, hook := range cfg.Hooks.PreDeploy {
				fmt.Printf("    - %s\n", hook)
			}
		}
		
		if len(cfg.Hooks.PostDeploy) > 0 {
			fmt.Printf("  %s:\n", t.DryRunPostDeploy)
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