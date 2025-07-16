package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
)

// version 变量将由 GoReleaser 注入。
var version string

// SetVersion 允许 main 包设置版本号。
func SetVersion(v string) {
	version = v
}

// NewUpdateCommand 创建 `revlay update` 命令。
func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "将 Revlay 更新到最新版本",
		Long:  "检查 GitHub Releases 并将 Revlay 程序原地更新到最新的可用版本。",
		RunE:  runUpdate,
	}
	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if version == "" {
		fmt.Println(color.Yellow("无法确定当前版本 (可能是开发版本)，无法进行更新。"))
		return nil
	}

	// 禁用库的默认日志输出，由我们自己控制
	selfupdate.EnableLog()
	log.SetOutput(&nullWriter{})

	v, err := semver.ParseTolerant(version)
	if err != nil {
		return fmt.Errorf("无法解析当前版本号 '%s': %w", version, err)
	}

	fmt.Printf("当前版本: %s\n", color.Cyan(v.String()))
	fmt.Println("正在检查更新...")

	latest, err := selfupdate.UpdateSelf(v, "xukonxe/Revlay")
	if err != nil {
		// 检查是否是权限错误
		if strings.Contains(err.Error(), "permission denied") {
			executable, _ := os.Executable()
			fmt.Println(color.Red("\n权限被拒绝。"))
			fmt.Println(color.Yellow("由于 Revlay 安装在受保护的目录中，您需要使用 'sudo' 来进行更新。"))
			fmt.Println(color.Yellow("请尝试运行以下命令:"))
			fmt.Printf("\n  %s\n\n", color.Cyan("sudo "+executable+" update"))
			return nil // 返回 nil，因为这是一个引导性的"错误"，而不是程序本身的 bug
		}
		return fmt.Errorf("更新检查失败: %w", err)
	}

	if latest.Version.Equals(v) {
		fmt.Println(color.Green("您使用的已经是最新版本！"))
	} else {
		fmt.Printf("已成功更新到版本: %s\n", color.Green(latest.Version.String()))
		fmt.Println("更新日志:")
		fmt.Println(latest.ReleaseNotes)
	}

	return nil
}

// nullWriter 用于丢弃 selfupdate 库的默认日志
type nullWriter struct{}

func (w *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
