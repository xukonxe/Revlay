package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
)

const (
	updateCheckInterval     = 24 * time.Hour // 检查间隔：24小时
	updateCheckTriggerEnv   = "REVLAY_UPDATE_CHECK"
	updateCheckTimestampEnv = "REVLAY_LAST_CHECK_TIMESTAMP"
)

type updateCache struct {
	LastCheckTimestamp int64 `json:"last_check_timestamp"`
}

// getCachePath 返回缓存文件的路径
func getCachePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "" // 在无法获取用户目录时静默失败
	}
	revlayConfigDir := filepath.Join(configDir, "revlay")
	return filepath.Join(revlayConfigDir, "update_cache.json")
}

// readCache 读取缓存文件
func readCache() *updateCache {
	path := getCachePath()
	if path == "" {
		return &updateCache{}
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &updateCache{}
	} else if err != nil {
		return &updateCache{}
	}
	var cache updateCache
	_ = json.Unmarshal(data, &cache)
	return &cache
}

// writeCache 写入缓存文件
func writeCache(cache *updateCache) {
	path := getCachePath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

// shouldTriggerUpdateCheck 判断是否应该启动一个后台检查进程
func shouldTriggerUpdateCheck() bool {
	// 如果是后台进程自己，或者版本未知，则不触发
	if os.Getenv(updateCheckTriggerEnv) != "" || version == "" {
		return false
	}
	// 如果是 update 或 version 命令，则不触发
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "update", "--version", "version":
			return false
		}
	}
	cache := readCache()
	return time.Since(time.Unix(cache.LastCheckTimestamp, 0)) > updateCheckInterval
}

// triggerBackgroundUpdateCheck 启动一个独立的后台进程来检查更新
func triggerBackgroundUpdateCheck() {
	cache := readCache()
	// 更新缓存时间戳，防止短时间内重复触发
	cache.LastCheckTimestamp = time.Now().Unix()
	writeCache(cache)

	// 获取当前可执行文件的路径
	exe, err := os.Executable()
	if err != nil {
		return
	}
	// 创建一个新命令
	cmd := exec.Command(exe, "__check_update")
	// 设置环境变量，标记这是一个后台检查进程
	cmd.Env = append(os.Environ(), updateCheckTriggerEnv+"=1")
	// 分离进程，让它在后台独立运行
	if err := cmd.Start(); err != nil {
		// 启动失败，静默处理
		return
	}
	// 主程序不等待，直接继续
	_ = cmd.Process.Release()
}

// NewCheckUpdateCommand 创建一个隐藏的命令，仅用于后台执行
func NewCheckUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "__check_update",
		Short:  "后台更新检查 (内部使用)",
		Hidden: true, // 在帮助信息中隐藏此命令
		Run: func(cmd *cobra.Command, args []string) {
			// 只检测，不更新
			latest, found, err := selfupdate.DetectLatest("xukonxe/Revlay")
			if err != nil || !found {
				return
			}

			currentV, err := semver.ParseTolerant(version)
			if err != nil {
				return
			}

			// 如果有新版本，则打印提示
			if latest.Version.GT(currentV) {
				// 使用 stderr 输出
				fmt.Fprintf(os.Stderr, "\n\n%s\n",
					color.Yellow("提示: Revlay 有一个新版本可用 (%s -> %s)。请运行 'revlay update' 进行更新。", version, latest.Version.String()),
				)
			}
		},
	}
	return cmd
}
