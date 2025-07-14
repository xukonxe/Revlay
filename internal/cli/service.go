package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/deployment"
)

// NewServiceCommand 创建服务管理命令
func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "管理 Revlay 服务",
		Long:  "管理 Revlay 服务列表，包括添加、删除和列出服务。",
	}

	// 添加子命令
	cmd.AddCommand(newServiceAddCommand())
	cmd.AddCommand(newServiceRemoveCommand())
	cmd.AddCommand(newServiceListCommand())

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
