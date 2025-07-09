package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/ssh"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [release-name]",
	Short: "Deploy a new release",
	Long: `Deploy a new release to the server.
	
If no release name is provided, a timestamp-based name will be generated.
This command will create a new release directory, link shared paths,
and switch the current symlink to the new release.`,
	RunE: runDeploy,
}

var (
	deployDryRun bool
)

func init() {
	deployCmd.Flags().BoolVarP(&deployDryRun, "dry-run", "d", false, "Show what would be done without actually deploying")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Generate release name if not provided
	var releaseName string
	if len(args) > 0 {
		releaseName = args[0]
	} else {
		releaseName = deployment.GenerateReleaseTimestamp()
	}

	fmt.Printf("🚀 Starting deployment of release: %s\n", releaseName)

	if deployDryRun {
		fmt.Println("🔍 DRY RUN MODE - No actual changes will be made")
		return runDeployDryRun(cfg, releaseName)
	}

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
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer client.Close()

	// Test connection
	fmt.Println("🔗 Testing SSH connection...")
	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("SSH connection test failed: %w", err)
	}
	fmt.Println("✓ SSH connection successful")

	// Create deployer
	deployer := deployment.NewDeployer(cfg, client)

	// Perform deployment
	fmt.Println("📦 Deploying release...")
	if err := deployer.Deploy(releaseName); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	fmt.Printf("✓ Deployment completed successfully\n")
	fmt.Printf("✓ Release %s is now live at %s\n", releaseName, cfg.Deploy.Path)

	return nil
}

func runDeployDryRun(cfg *config.Config, releaseName string) error {
	fmt.Println("📋 Deployment plan:")
	fmt.Printf("  - Application: %s\n", cfg.App.Name)
	fmt.Printf("  - Server: %s@%s:%d\n", cfg.Server.User, cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  - Release: %s\n", releaseName)
	fmt.Printf("  - Deploy path: %s\n", cfg.Deploy.Path)
	fmt.Printf("  - Releases path: %s\n", cfg.GetReleasesPath())
	fmt.Printf("  - Shared path: %s\n", cfg.GetSharedPath())
	fmt.Printf("  - Current path: %s\n", cfg.GetCurrentPath())
	fmt.Printf("  - Release path: %s\n", cfg.GetReleasePathByName(releaseName))
	
	fmt.Println("\n📂 Directory structure to be created:")
	fmt.Printf("  %s/\n", cfg.Deploy.Path)
	fmt.Printf("  ├── releases/\n")
	fmt.Printf("  │   └── %s/\n", releaseName)
	fmt.Printf("  ├── shared/\n")
	for _, sharedPath := range cfg.Deploy.SharedPaths {
		fmt.Printf("  │   └── %s\n", sharedPath)
	}
	fmt.Printf("  └── current -> releases/%s\n", releaseName)

	fmt.Println("\n🔗 Shared paths to be linked:")
	for _, sharedPath := range cfg.Deploy.SharedPaths {
		fmt.Printf("  - %s\n", sharedPath)
	}

	fmt.Println("\n🪝 Hooks to be executed:")
	if len(cfg.Hooks.PreDeploy) > 0 {
		fmt.Println("  Pre-deploy:")
		for _, hook := range cfg.Hooks.PreDeploy {
			fmt.Printf("    - %s\n", hook)
		}
	}
	if len(cfg.Hooks.PostDeploy) > 0 {
		fmt.Println("  Post-deploy:")
		for _, hook := range cfg.Hooks.PostDeploy {
			fmt.Printf("    - %s\n", hook)
		}
	}

	fmt.Printf("\n🧹 Keep %d releases (older ones will be cleaned up)\n", cfg.App.KeepReleases)

	return nil
}