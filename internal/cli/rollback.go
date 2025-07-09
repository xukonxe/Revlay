package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/ssh"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [release-name]",
	Short: "Rollback to a previous release",
	Long: `Rollback to a previous release.
	
If no release name is provided, it will rollback to the previous release.
This command will switch the current symlink to point to the specified release.`,
	RunE: runRollback,
}

func runRollback(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("failed to connect to server: %w", err)
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

	fmt.Printf("🔄 Rolling back to release: %s\n", targetRelease)

	// Perform rollback
	if err := deployer.Rollback(targetRelease); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf("✓ Rollback completed successfully\n")
	fmt.Printf("✓ Release %s is now live at %s\n", targetRelease, cfg.Deploy.Path)

	return nil
}