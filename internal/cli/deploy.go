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

// NewDeployCommand 创建部署命令
func NewDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [release-name]",
		Short: i18n.T().DeployShortDesc,
		Long:  i18n.T().DeployLongDesc,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 获取标志
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			fromDir, _ := cmd.Flags().GetString("from-dir")
			beautify, _ := cmd.Flags().GetBool("beautify") // 获取美化界面标志

			// 处理 --app 参数
			cfgFile, err := resolveAppConfig(cmd)
			if err != nil {
				fmt.Println(color.Red("Error: %v", err))
				return
			}

			cfg, err := loadConfig(cfgFile)
			if err != nil {
				fmt.Println(color.Red("Error: %v", err))
				return
			}

			// 当用户在项目目录中直接运行 `revlay deploy` 时，
			// 自动检查并添加服务到全局列表（如果尚未添加）
			appID, _ := cmd.Flags().GetString("app")
			if appID == "" {
				allServices, err := config.ListServices()
				if err != nil {
					// 此处不返回错误，仅打印警告，因为这不应阻塞核心的部署功能
					fmt.Println(color.Yellow("警告: 无法检查全局服务列表: %v", err))
				} else {
					isRegistered := false
					for _, service := range allServices {
						// 通过比较根目录来判断服务是否已注册
						if service.Root == cfg.RootPath {
							isRegistered = true
							break
						}
					}

					if !isRegistered {
						appName := cfg.App.Name
						fmt.Printf("服务 '%s' 尚未在全局列表中注册，正在尝试自动添加...\n", appName)
						// 使用应用的名称作为全局唯一的 ID
						if err := config.AddService(appName, appName, cfg.RootPath); err != nil {
							fmt.Println(color.Yellow("警告: 自动添加服务失败: %v", err))
							fmt.Println(color.Yellow("你可以稍后手动添加，例如: revlay service add %s .", appName))
						} else {
							fmt.Println(color.Green("服务 '%s' 已成功添加到全局列表。", appName))
						}
					}
				}
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
				runDeployDryRun(cfg, releaseName)
				return
			}

			// 使用美化选项创建部署器
			var deployer deployment.Deployer
			if beautify {
				deployer = deployment.NewLocalDeployerWithOptions(cfg, true)
			} else {
				deployer = deployment.NewLocalDeployer(cfg)
			}

			fmt.Println(color.Cyan(i18n.T().DeployInProgress))
			if err := deployer.Deploy(releaseName, fromDir); err != nil {
				fmt.Println(color.Red(i18n.T().DeployFailed, err))
				return
			}

			fmt.Println(color.Green(i18n.T().DeploySuccess))
			fmt.Printf(i18n.T().DeployReleaseLive, releaseName, cfg.RootPath)
		},
	}

	// 标准的 deploy 标志
	cmd.Flags().BoolP("dry-run", "d", false, i18n.T().DeployDryRunFlag)
	cmd.Flags().String("from-dir", "", i18n.T().DeployFromDirFlag)
	cmd.Flags().StringP("app", "a", "", "指定要部署的服务 ID（从全局服务列表中）")

	// 添加美化界面选项
	cmd.Flags().Bool("beautify", false, "使用美化输出界面")

	return cmd
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
