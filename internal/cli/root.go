package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
)

// Execute is the main entry point for the CLI.
func Execute() {
	// 首先手动解析命令行参数中的语言标志
	langFlag := ""
	for i, arg := range os.Args {
		if arg == "--lang" && i+1 < len(os.Args) {
			langFlag = os.Args[i+1]
			break
		} else if strings.HasPrefix(arg, "--lang=") {
			langFlag = strings.TrimPrefix(arg, "--lang=")
			break
		} else if strings.HasPrefix(arg, "-l") && len(arg) == 2 && i+1 < len(os.Args) {
			langFlag = os.Args[i+1]
			break
		}
	}

	// 初始化语言
	i18n.InitLanguage(langFlag)

	// 创建根命令
	rootCmd := newRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.Red("Error: %v", err))
		os.Exit(1)
	}
}

// newRootCmd creates the root command and adds all subcommands.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revlay",
		Short:   i18n.T().AppShortDesc,
		Long:    i18n.T().AppLongDesc,
		Version: version, // 这个 'version' 变量由 main.go 传入
		// Silence errors, we'll handle them in Execute()
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Cobra 会自动处理 --version 标志的逻辑，
	// 但我们也可以显式地设置模板，使其更美观
	cmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	// Add all the commands
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewDeployCommand())
	cmd.AddCommand(NewRollbackCommand())
	cmd.AddCommand(NewReleasesCommand())
	cmd.AddCommand(NewStatusCommand())
	cmd.AddCommand(NewPushCommand())
	cmd.AddCommand(NewProxyCommand())   // Add the new proxy command
	cmd.AddCommand(NewServiceCommand()) // 添加服务管理命令
	cmd.AddCommand(NewPsCommand())      // 添加 ps 命令作为 service list 的别名
	cmd.AddCommand(NewStartCommand())   // 添加 start 命令作为 service start 的别名
	cmd.AddCommand(NewStopCommand())    // 添加 stop 命令作为 service stop 的别名
	cmd.AddCommand(NewUpdateCommand())
	cmd.AddCommand(NewCheckUpdateCommand())

	// Add persistent flags to the root command.
	cmd.PersistentFlags().StringP("config", "c", "", i18n.T().ConfigFileFlag)
	cmd.PersistentFlags().StringP("lang", "l", "", i18n.T().LanguageFlag)

	// 在这里添加所有命令...
	// ...

	// 6. 在所有命令都添加完毕后，再检查是否需要启动后台更新
	if shouldTriggerUpdateCheck() {
		triggerBackgroundUpdateCheck()
	}

	return cmd
}

// NewStartCommand 创建 start 命令作为 service start 的别名
func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [id]",
		Short: i18n.T().ServiceStartShortDesc,
		Long:  i18n.T().ServiceStartLongDesc + " 这是 'service start' 命令的别名。",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 直接调用 service start 命令的逻辑
			serviceCmd := NewServiceCommand()
			startCmd := NewServiceStartCommand()
			serviceCmd.AddCommand(startCmd)
			startCmd.SetArgs(args)
			return startCmd.Execute()
		},
	}
	return cmd
}

// NewStopCommand 创建 stop 命令作为 service stop 的别名
func NewStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [id]",
		Short: i18n.T().ServiceStopShortDesc,
		Long:  i18n.T().ServiceStopLongDesc + " 这是 'service stop' 命令的别名。",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 直接调用 service stop 命令的逻辑
			serviceCmd := NewServiceCommand()
			stopCmd := NewServiceStopCommand()
			serviceCmd.AddCommand(stopCmd)
			stopCmd.SetArgs(args)
			return stopCmd.Execute()
		},
	}
	return cmd
}
