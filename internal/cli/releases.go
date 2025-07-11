package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/color"
)

// NewReleasesCommand creates the `revlay releases` command.
func NewReleasesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "releases",
		Short: "Lists all available releases",
		Long:  `Scans the releases directory and lists all previously deployed release versions.`,
		RunE:  runReleases,
	}
	return cmd
}

func runReleases(cmd *cobra.Command, args []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	deployer := deployment.NewLocalDeployer(cfg)
	releases, err := deployer.ListReleases()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	if len(releases) == 0 {
		fmt.Println("No releases found.")
		return nil
	}

	currentRelease, _ := deployer.GetCurrentRelease()

	fmt.Printf("Available releases for '%s':\n", color.Cyan(cfg.App.Name))
	for _, release := range releases {
		if release == currentRelease {
			fmt.Printf("  - %s %s\n", color.Green(release), color.Yellow("(current)"))
		} else {
			fmt.Printf("  - %s\n", release)
		}
	}

	return nil
}