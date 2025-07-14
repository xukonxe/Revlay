package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/config"
)

// resolveAppConfig 处理 --app 参数，如果指定了 app，则从全局服务列表中获取服务配置
// 返回应该使用的配置文件路径
func resolveAppConfig(cmd *cobra.Command) (string, error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	appID, _ := cmd.Flags().GetString("app")

	// 如果指定了 app，则从全局服务列表中获取服务配置
	if appID != "" {
		service, err := config.GetService(appID)
		if err != nil {
			return "", fmt.Errorf("获取服务失败: %w", err)
		}

		// 使用服务的根目录作为工作目录
		if err := os.Chdir(service.Root); err != nil {
			return "", fmt.Errorf("切换到服务目录失败: %w", err)
		}

		// 使用服务目录中的配置文件
		cfgFile = filepath.Join(service.Root, "revlay.yml")
		fmt.Printf("使用服务 '%s' (%s) 的配置文件: %s\n", appID, service.Name, cfgFile)
	}

	return cfgFile, nil
}
