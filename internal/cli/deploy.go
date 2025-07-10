package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ssh"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [release-name]",
	Short: "",
	Long:  ``,
	RunE: runDeploy,
}

var (
	deployDryRun bool
)

func init() {
	deployCmd.Flags().BoolVarP(&deployDryRun, "dry-run", "d", false, "")
	
	// Update command descriptions when config is initialized
	cobra.OnInitialize(func() {
		t := i18n.T()
		deployCmd.Short = t.DeployShortDesc
		deployCmd.Long = t.DeployLongDesc
		deployCmd.Flags().Lookup("dry-run").Usage = t.DeployDryRunFlag
	})
}

func runDeploy(cmd *cobra.Command, args []string) error {
	t := i18n.T()
	
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

	fmt.Printf(t.DeployStarting+"\n", releaseName)

	if deployDryRun {
		fmt.Println(t.DeployDryRunMode)
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
		return fmt.Errorf(t.ErrorSSHConnect, err)
	}
	defer client.Close()

	// Test connection
	fmt.Println(t.DeploySSHTest)
	if err := client.TestConnection(); err != nil {
		return fmt.Errorf(t.ErrorSSHTest, err)
	}
	fmt.Println(t.DeploySSHSuccess)

	// Create deployer
	deployer := deployment.NewDeployer(cfg, client)

	// Perform deployment based on mode
	fmt.Println(t.DeployInProgress)
	if err := performDeployment(deployer, cfg, releaseName); err != nil {
		return fmt.Errorf(t.ErrorDeployment, err)
	}

	fmt.Println(t.DeploySuccess)
	fmt.Printf(t.DeployReleaseLive+"\n", releaseName, cfg.Deploy.Path)

	return nil
}

func performDeployment(deployer *deployment.Deployer, cfg *config.Config, releaseName string) error {
	switch cfg.Deploy.Mode {
	case config.ZeroDowntimeMode:
		return deployer.DeployZeroDowntime(releaseName)
	case config.ShortDowntimeMode:
		return deployer.Deploy(releaseName)
	default:
		return deployer.Deploy(releaseName)
	}
}

func runDeployDryRun(cfg *config.Config, releaseName string) error {
	t := i18n.T()
	
	fmt.Println(t.DryRunPlan)
	fmt.Printf("  - %s: %s\n", t.DryRunApplication, cfg.App.Name)
	fmt.Printf("  - %s: %s@%s:%d\n", t.DryRunServer, cfg.Server.User, cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  - %s: %s\n", t.DryRunRelease, releaseName)
	fmt.Printf("  - %s: %s\n", t.DryRunDeployPath, cfg.Deploy.Path)
	fmt.Printf("  - %s: %s\n", t.DryRunReleasesPath, cfg.GetReleasesPath())
	fmt.Printf("  - %s: %s\n", t.DryRunSharedPath, cfg.GetSharedPath())
	fmt.Printf("  - %s: %s\n", t.DryRunCurrentPath, cfg.GetCurrentPath())
	fmt.Printf("  - %s: %s\n", t.DryRunReleasePathFmt, cfg.GetReleasePathByName(releaseName))
	fmt.Printf("  - %s: %s\n", t.DeploymentMode, cfg.Deploy.Mode)
	
	fmt.Println("\n" + t.DryRunDirStructure)
	fmt.Printf("  %s/\n", cfg.Deploy.Path)
	fmt.Printf("  ├── releases/\n")
	fmt.Printf("  │   └── %s/\n", releaseName)
	fmt.Printf("  ├── shared/\n")
	for _, sharedPath := range cfg.Deploy.SharedPaths {
		fmt.Printf("  │   └── %s\n", sharedPath)
	}
	fmt.Printf("  └── current -> releases/%s\n", releaseName)

	fmt.Println("\n" + t.DryRunSharedPaths)
	for _, sharedPath := range cfg.Deploy.SharedPaths {
		fmt.Printf("  - %s\n", sharedPath)
	}

	fmt.Println("\n" + t.DryRunHooks)
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

	// Show deployment mode specific information
	if cfg.Deploy.Mode == config.ZeroDowntimeMode {
		fmt.Printf("\n%s (%s):\n", t.DeploymentMode, t.ZeroDowntime)
		fmt.Printf("  - %s: %d\n", t.ServicePort, cfg.Service.Port)
		fmt.Printf("  - Alternative Port: %d\n", cfg.Service.AltPort)
		fmt.Printf("  - %s: %s\n", t.ServiceHealthCheck, cfg.Service.HealthCheck)
		fmt.Printf("  - %s: %ds\n", t.ServiceRestartDelay, cfg.Service.RestartDelay)
	} else {
		fmt.Printf("\n%s (%s):\n", t.DeploymentMode, t.ShortDowntime)
		fmt.Printf("  - %s: %s\n", t.ServiceCommand, cfg.Service.Command)
		fmt.Printf("  - Graceful Timeout: %ds\n", cfg.Service.GracefulTimeout)
	}

	fmt.Printf("\n"+t.DryRunKeepReleases+"\n", cfg.App.KeepReleases)

	return nil
}