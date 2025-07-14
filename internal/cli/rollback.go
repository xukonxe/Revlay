package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
)

// NewRollbackCommand creates the `revlay rollback` command.
func NewRollbackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback [release-name]",
		Short: i18n.T().RollbackShortDesc,
		Long:  i18n.T().RollbackLongDesc,
		RunE:  runRollback,
	}
	cmd.Flags().StringP("app", "a", "", "指定要回滚的服务 ID（从全局服务列表中）")
	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
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

	var releaseName string
	if len(args) > 0 {
		releaseName = args[0]
	}

	// If no release name is given, rollback to the previous one
	if releaseName == "" {
		releases, err := deployer.ListReleases()
		if err != nil {
			return fmt.Errorf(i18n.T().RollbackFailed, fmt.Sprintf("could not list releases to determine previous version: %v", err))
		}
		if len(releases) < 2 {
			return fmt.Errorf(i18n.T().RollbackFailed, i18n.T().RollbackNoReleases)
		}
		releaseName = releases[len(releases)-2] // The second to last one
	}

	fmt.Printf(i18n.T().RollbackToRelease, color.Yellow(releaseName))
	fmt.Println()

	if err := deployer.Rollback(releaseName); err != nil {
		return fmt.Errorf(i18n.T().RollbackFailed, err)
	}

	fmt.Println(color.Green(i18n.T().RollbackSuccess, releaseName))
	return nil
}
