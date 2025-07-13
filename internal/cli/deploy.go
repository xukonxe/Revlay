package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
)

func NewDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [release-name]",
		Short: i18n.T().DeployShortDesc,
		Long:  i18n.T().DeployLongDesc,
		RunE:  runDeploy,
	}

	cmd.Flags().BoolP("dry-run", "d", false, i18n.T().DeployDryRunFlag)
	cmd.Flags().String("from-dir", "", i18n.T().DeployFromDirFlag)

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

	fmt.Println(color.Green(i18n.T().DeployStarting, releaseName))

	if dryRun {
		fmt.Println(color.Yellow(i18n.T().DeployDryRunMode))
		return runDeployDryRun(cfg, releaseName)
	}

	deployer := deployment.NewLocalDeployer(cfg)

	fmt.Println(color.Cyan(i18n.T().DeployInProgress))
	if err := deployer.Deploy(releaseName, fromDir); err != nil {
		return fmt.Errorf(i18n.T().DeployFailed, err)
	}

	fmt.Println(color.Green(i18n.T().DeploySuccess))
	fmt.Printf(i18n.T().DeployReleaseLive, releaseName, cfg.RootPath)

	return nil
}

func runDeployDryRun(cfg *config.Config, releaseName string) error {
	fmt.Println(i18n.T().DryRunPlan)
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunApplication, cfg.App.Name)
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunRelease, releaseName)
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunDeployPath, cfg.RootPath)
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunReleasesPath, cfg.GetReleasesPath())
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunSharedPath, cfg.GetSharedPath())
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunCurrentPath, cfg.GetCurrentPath())
	fmt.Printf("  - %s: %s\n", i18n.T().DryRunReleasePathFmt, cfg.GetReleasePathByName(releaseName))

	fmt.Println("\n" + i18n.T().DryRunDirStructure)
	fmt.Printf("  %s/\n", filepath.Base(cfg.RootPath))
	fmt.Printf("  ├── releases/\n")
	fmt.Printf("  │   └── %s/ (new release directory)\n", releaseName)
	fmt.Printf("  ├── shared/\n")
	fmt.Printf("  └── current -> releases/%s (atomic symlink switch)\n", releaseName)

	fmt.Println("\n" + i18n.T().DryRunHooks + ":")
	if len(cfg.Hooks.PreDeploy) > 0 {
		fmt.Println("  " + i18n.T().DryRunPreDeploy + ":")
		for _, hook := range cfg.Hooks.PreDeploy {
			fmt.Printf("    - %s\n", hook)
		}
	}
	if len(cfg.Hooks.PostDeploy) > 0 {
		fmt.Println("  " + i18n.T().DryRunPostDeploy + ":")
		for _, hook := range cfg.Hooks.PostDeploy {
			fmt.Printf("    - %s\n", hook)
		}
	}

	fmt.Printf("\n" + i18n.Sprintf(i18n.T().DryRunKeepReleases, cfg.App.KeepReleases) + "\n")

	return nil
}
