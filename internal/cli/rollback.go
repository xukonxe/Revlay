package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ssh"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [release-name]",
	Short: "",
	Long:  ``,
	RunE: runRollback,
}

func init() {
	// Update command descriptions when config is initialized
	cobra.OnInitialize(func() {
		t := i18n.T()
		rollbackCmd.Short = t.RollbackShortDesc
		rollbackCmd.Long = t.RollbackLongDesc
	})
}

func runRollback(cmd *cobra.Command, args []string) error {
	t := i18n.T()
	
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
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

	// Create deployer
	deployer := deployment.NewDeployer(cfg, client)

	// Determine target release
	var targetRelease string
	if len(args) > 0 {
		targetRelease = args[0]
	} else {
		// Get previous release
		releases, err := deployer.ListReleases()
		if err != nil {
			return fmt.Errorf("failed to list releases: %w", err)
		}

		if len(releases) < 2 {
			return fmt.Errorf("no previous release available for rollback")
		}

		// Find current release index
		currentIndex := -1
		for i, release := range releases {
			if release.Current {
				currentIndex = i
				break
			}
		}

		if currentIndex == -1 {
			return fmt.Errorf("no current release found")
		}

		if currentIndex >= len(releases)-1 {
			return fmt.Errorf("no previous release available for rollback")
		}

		targetRelease = releases[currentIndex+1].Name
	}

	fmt.Printf(t.RollbackToRelease+"\n", targetRelease)

	// Perform rollback
	if err := deployer.Rollback(targetRelease); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf(t.RollbackSuccess+"\n", targetRelease)
	fmt.Printf(t.DeployReleaseLive+"\n", targetRelease, cfg.Deploy.Path)

	return nil
}