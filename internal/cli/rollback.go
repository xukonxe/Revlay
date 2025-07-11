package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/color"
)

// NewRollbackCommand creates the `revlay rollback` command.
func NewRollbackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback [release-name]",
		Short: "Rolls back to a previous release",
		Long: `Switches the 'current' symlink to a specified previous release. 
If no release name is provided, it rolls back to the second to last release.`,
		RunE: runRollback,
	}
	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	deployer := deployment.NewLocalDeployer(cfg)

	var releaseName string
	if len(args) > 0 {
		releaseName = args[0]
	}

	// If no release name is given, rollback to the previous one
	if releaseName == "" {
		releases, err := deployer.ListReleases()
		if err != nil {
			return fmt.Errorf("could not list releases to determine previous version: %w", err)
		}
		if len(releases) < 2 {
			return fmt.Errorf("not enough releases to roll back to. At least two are required")
		}
		releaseName = releases[len(releases)-2] // The second to last one
	}

	fmt.Printf("🔥 Rolling back to release: %s\n", color.Yellow(releaseName))

	if err := deployer.Rollback(releaseName); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Println(color.Green("✅ Rollback successful!"))
	fmt.Printf("   - Release '%s' is now live.\n", releaseName)
	return nil
}