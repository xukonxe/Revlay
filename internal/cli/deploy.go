package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/color"
)

func NewDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [release-name]",
		Short: "Deploys a new release to the application directory",
		Long: `This command executes the deployment process based on the settings in revlay.yml.
It creates a new release directory, runs deployment hooks, and atomically switches the 'current' symlink.`,
		RunE: runDeploy,
	}

	cmd.Flags().BoolP("dry-run", "d", false, "Simulate deployment without making any changes")
	cmd.Flags().String("from-dir", "", "Deploy from a specific directory instead of an empty one")

	return cmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	fromDir, _ := cmd.Flags().GetString("from-dir")

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	var releaseName string
	if len(args) > 0 {
		releaseName = args[0]
	} else {
		releaseName = deployment.GenerateReleaseTimestamp()
	}

	fmt.Println(color.Green("🚀 Starting deployment of release: %s", releaseName))

	if dryRun {
		fmt.Println(color.Yellow("== Dry Run Mode: No changes will be made =="))
		return runDeployDryRun(cfg, releaseName)
	}

	deployer := deployment.NewLocalDeployer(cfg)

	fmt.Println(color.Cyan("   - Deployment in progress..."))
	if err := deployer.Deploy(releaseName, fromDir); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	fmt.Println(color.Green("✅ Deployment successful!"))
	fmt.Printf("   - Release '%s' is now live in '%s'\n", releaseName, cfg.RootPath)

	return nil
}

func runDeployDryRun(cfg *config.Config, releaseName string) error {
	fmt.Println("Deployment Plan:")
	fmt.Printf("  - Application: %s\n", cfg.App.Name)
	fmt.Printf("  - Release: %s\n", releaseName)
	fmt.Printf("  - Deploy Path: %s\n", cfg.RootPath)
	fmt.Printf("  - Releases Path: %s\n", cfg.GetReleasesPath())
	fmt.Printf("  - Shared Path: %s\n", cfg.GetSharedPath())
	fmt.Printf("  - Current Path: %s\n", cfg.GetCurrentPath())
	fmt.Printf("  - Path for this release: %s\n", cfg.GetReleasePathByName(releaseName))

	fmt.Println("\nDirectory Structure:")
	fmt.Printf("  %s/\n", filepath.Base(cfg.RootPath))
	fmt.Printf("  ├── releases/\n")
	fmt.Printf("  │   └── %s/ (new release directory)\n", releaseName)
	fmt.Printf("  ├── shared/\n")
	fmt.Printf("  └── current -> releases/%s (atomic symlink switch)\n", releaseName)

	fmt.Println("\nHooks:")
	if len(cfg.Hooks.PreDeploy) > 0 {
		fmt.Println("  Pre-Deploy:")
		for _, hook := range cfg.Hooks.PreDeploy {
			fmt.Printf("    - %s\n", hook)
		}
	}
	if len(cfg.Hooks.PostDeploy) > 0 {
		fmt.Println("  Post-Deploy:")
		for _, hook := range cfg.Hooks.PostDeploy {
			fmt.Printf("    - %s\n", hook)
		}
	}

	fmt.Printf("\nKeep Releases: %d\n", cfg.App.KeepReleases)
	
	return nil
}