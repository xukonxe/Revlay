package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
)

// NewServiceCommand 创建服务管理命令
func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: i18n.T().ServiceShortDesc,
		Long:  i18n.T().ServiceLongDesc,
	}

	// 添加子命令
	cmd.AddCommand(newServiceAddCommand())
	cmd.AddCommand(newServiceRemoveCommand())
	cmd.AddCommand(newServiceListCommand())
	cmd.AddCommand(NewServiceStartCommand())
	cmd.AddCommand(NewServiceStopCommand())

	return cmd
}

// newServiceAddCommand 创建添加服务的命令
func newServiceAddCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "add [id] [path]",
		Short: "添加一个服务到全局服务列表",
		Long:  "添加一个服务到全局服务列表，需要指定服务 ID 和路径。",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			path, err := filepath.Abs(args[1])
			if err != nil {
				return fmt.Errorf("无法获取绝对路径: %w", err)
			}

			// 如果未指定名称，则使用 ID 作为名称
			if name == "" {
				name = id
			}

			// 检查路径是否存在
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("路径 '%s' 不存在", path)
			}

			// 检查路径中是否存在 revlay.yml 文件
			revlayConfigPath := filepath.Join(path, "revlay.yml")
			if _, err := os.Stat(revlayConfigPath); os.IsNotExist(err) {
				return fmt.Errorf("在 '%s' 中未找到 revlay.yml 文件", path)
			}

			// 添加服务
			if err := config.AddService(id, name, path); err != nil {
				return fmt.Errorf("添加服务失败: %w", err)
			}

			fmt.Printf("服务 '%s' 已成功添加到全局服务列表。\n", id)
			return nil
		},
	}

	// 添加标志
	cmd.Flags().StringVarP(&name, "name", "n", "", "服务的显示名称（默认与 ID 相同）")

	return cmd
}

// newServiceRemoveCommand 创建删除服务的命令
func newServiceRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [id]",
		Short: "从全局服务列表中移除一个服务",
		Long:  "从全局服务列表中移除一个服务，需要指定服务 ID。",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// 移除服务
			if err := config.RemoveService(id); err != nil {
				return fmt.Errorf("移除服务失败: %w", err)
			}

			fmt.Printf("服务 '%s' 已成功从全局服务列表中移除。\n", id)
			return nil
		},
	}

	return cmd
}

// newServiceListCommand 创建列出服务的命令
func newServiceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出全局服务列表中的所有服务",
		Long:  "列出全局服务列表中的所有服务，包括它们的 ID、名称和路径。",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取所有服务
			services, err := config.ListServices()
			if err != nil {
				return fmt.Errorf("获取服务列表失败: %w", err)
			}

			if len(services) == 0 {
				fmt.Println("全局服务列表为空。使用 'revlay service add' 添加服务。")
				return nil
			}

			// 创建一个有序的服务 ID 列表，以便按字母顺序显示
			var serviceIDs []string
			for id := range services {
				serviceIDs = append(serviceIDs, id)
			}
			sort.Strings(serviceIDs)

			// 使用 tabwriter 格式化输出
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\t名称\t路径\t当前版本")
			fmt.Fprintln(w, "----\t----\t----\t----")

			for _, id := range serviceIDs {
				service := services[id]

				// 尝试获取当前版本
				currentVersion := "未部署"
				cfg, err := config.LoadConfig(filepath.Join(service.Root, "revlay.yml"))
				if err == nil {
					cfg.RootPath = service.Root
					deployer := deployment.NewLocalDeployer(cfg)
					if release, err := deployer.GetCurrentRelease(); err == nil && release != "" {
						currentVersion = release
					}
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, service.Name, service.Root, currentVersion)
			}
			w.Flush()

			return nil
		},
	}

	return cmd
}

// NewServiceStartCommand 创建启动服务的命令
func NewServiceStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [id]",
		Short: i18n.T().ServiceStartShortDesc,
		Long:  i18n.T().ServiceStartLongDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// 获取服务信息
			service, err := config.GetService(id)
			if err != nil {
				return fmt.Errorf(i18n.T().ServiceNotFound, id)
			}

			// 加载服务配置
			cfg, err := config.LoadConfig(filepath.Join(service.Root, "revlay.yml"))
			if err != nil {
				return fmt.Errorf("加载服务配置失败: %w", err)
			}
			cfg.RootPath = service.Root

			deployer := deployment.NewLocalDeployer(cfg)

			// 检查是否有部署的版本
			releaseName, err := deployer.GetCurrentRelease()
			if err != nil || releaseName == "" {
				return fmt.Errorf(i18n.T().ServiceNoReleaseFound, id)
			}

			// 启动服务
			fmt.Println(color.Cyan(i18n.Sprintf(i18n.T().ServiceStarting, id)))

			// 检查服务是否配置了启动命令
			if cfg.Service.StartCommand == "" {
				return fmt.Errorf(i18n.T().ServiceStartNotConfigured, id)
			}

			// 启动服务
			err = deployer.StartService(releaseName)
			var alreadyRunningErr *deployment.ServiceAlreadyRunningError
			if errors.As(err, &alreadyRunningErr) {
				fmt.Println(color.Yellow(i18n.Sprintf(i18n.T().ServiceAlreadyRunning, id, alreadyRunningErr.PID)))
				return nil
			} else if err != nil {
				return fmt.Errorf(i18n.T().ServiceStartFailed, id, err)
			}

			// 获取进程ID
			pidPath := filepath.Join(cfg.RootPath, "pids", cfg.App.Name+".pid")
			pidData, err := os.ReadFile(pidPath)
			if err != nil {
				fmt.Println(color.Green(i18n.Sprintf("服务 '%s' 已启动，但无法读取进程ID。", id)))
				return nil
			}

			parts := strings.Split(strings.TrimSpace(string(pidData)), ":")
			if len(parts) != 2 {
				fmt.Println(color.Green(i18n.Sprintf("服务 '%s' 已启动，但PID文件格式无效。", id)))
				return nil
			}

			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				fmt.Println(color.Green(i18n.Sprintf("服务 '%s' 已启动，但无法解析进程ID。", id)))
				return nil
			}

			fmt.Println(color.Green(i18n.Sprintf(i18n.T().ServiceStartSuccess, id, pid)))
			return nil
		},
	}

	return cmd
}

// NewServiceStopCommand 创建停止服务的命令
func NewServiceStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [id]",
		Short: i18n.T().ServiceStopShortDesc,
		Long:  i18n.T().ServiceStopLongDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// 获取服务信息
			service, err := config.GetService(id)
			if err != nil {
				return fmt.Errorf(i18n.T().ServiceNotFound, id)
			}

			// 加载服务配置
			cfg, err := config.LoadConfig(filepath.Join(service.Root, "revlay.yml"))
			if err != nil {
				return fmt.Errorf("加载服务配置失败: %w", err)
			}
			cfg.RootPath = service.Root

			deployer := deployment.NewLocalDeployer(cfg)

			// 检查服务是否已部署
			_, err = deployer.GetCurrentRelease()
			if err != nil {
				return fmt.Errorf(i18n.T().ServiceNoReleaseFound, id)
			}

			// 停止服务
			fmt.Println(color.Cyan(i18n.Sprintf(i18n.T().ServiceStopping, id)))

			// 检查PID文件是否存在
			pidPath := filepath.Join(cfg.RootPath, "pids", cfg.App.Name+".pid")
			if _, err := os.Stat(pidPath); os.IsNotExist(err) {
				fmt.Println(color.Yellow(i18n.Sprintf(i18n.T().ServiceStopNotRunning, id)))
				return nil
			}

			// 停止服务
			if err := deployer.StopService(); err != nil {
				return fmt.Errorf(i18n.T().ServiceStopFailed, id, err)
			}

			fmt.Println(color.Green(i18n.Sprintf(i18n.T().ServiceStopSuccess, id)))
			return nil
		},
	}

	return cmd
}

// NewPsCommand 创建一个 ps 命令作为 service list 的别名
func NewPsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "列出全局服务列表中的所有服务（service list 的别名）",
		Long:  "列出全局服务列表中的所有服务，包括它们的 ID、名称和路径。这是 'service list' 命令的别名。",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 直接调用 service list 命令的逻辑
			listCmd := newServiceListCommand()
			return listCmd.RunE(listCmd, args)
		},
	}

	return cmd
}
