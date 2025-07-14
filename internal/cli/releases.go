package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
)

// NewReleasesCommand creates the `revlay releases` command.
func NewReleasesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "releases",
		Short: i18n.T().ReleasesShortDesc,
		Long:  i18n.T().ReleasesLongDesc,
		RunE:  runReleases,
	}
	cmd.Flags().StringP("app", "a", "", "指定要查看的服务 ID（从全局服务列表中）")
	return cmd
}

func runReleases(cmd *cobra.Command, args []string) error {
	// 处理 --app 参数
	cfgFile, err := resolveAppConfig(cmd)
	if err != nil {
		return err
	}

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	deployer := deployment.NewLocalDeployer(cfg)
	releases, err := deployer.ListReleases()
	if err != nil {
		return fmt.Errorf(i18n.T().ErrorReleasesList, err)
	}

	if len(releases) == 0 {
		fmt.Println(i18n.T().ReleasesNoReleases)
		return nil
	}

	currentRelease, _ := deployer.GetCurrentRelease()

	fmt.Println(i18n.T().ReleasesListHeader)
	for _, release := range releases {
		if release == currentRelease {
			fmt.Printf("  - %s%s\n", color.Green(release), color.Yellow(i18n.T().ReleasesCurrent))
		} else {
			fmt.Printf("  - %s\n", release)
		}
	}

	return nil
}
