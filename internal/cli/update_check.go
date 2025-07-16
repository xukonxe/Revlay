package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/xukonxe/revlay/internal/color"
)

const (
	updateCheckInterval = 24 * time.Hour // 检查间隔：24小时
)

type updateCache struct {
	LastCheckTimestamp int64 `json:"last_check_timestamp"`
}

// getCachePath 返回缓存文件的路径
func getCachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	revlayConfigDir := filepath.Join(configDir, "revlay")
	if err := os.MkdirAll(revlayConfigDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(revlayConfigDir, "update_cache.json"), nil
}

// readCache 读取缓存文件
func readCache() (*updateCache, error) {
	path, err := getCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &updateCache{}, nil // 文件不存在是正常情况
	} else if err != nil {
		return nil, err
	}

	var cache updateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// writeCache 写入缓存文件
func writeCache(cache *updateCache) error {
	path, err := getCachePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// isCheckDue 判断是否需要进行更新检查
func isCheckDue() bool {
	cache, err := readCache()
	if err != nil {
		return true // 如果读缓存失败，就检查一次
	}
	return time.Since(time.Unix(cache.LastCheckTimestamp, 0)) > updateCheckInterval
}

// CheckForUpdatesAsync 检查更新，并优雅地提示用户。
// 这个函数本身是同步的，应该在一个 goroutine 中调用。
func CheckForUpdatesAsync() {
	// 如果当前版本未知（开发版），或不到检查时间，则直接返回
	if version == "" || !isCheckDue() {
		return
	}

	// 立即更新缓存中的最后检查时间，无论成功与否，避免频繁检查
	cache, _ := readCache()
	cache.LastCheckTimestamp = time.Now().Unix()
	_ = writeCache(cache)

	// 只检测，不更新
	latest, found, err := selfupdate.DetectLatest("xukonxe/Revlay")
	if err != nil || !found {
		// 网络错误等，静默失败，不打扰用户
		return
	}

	// 解析当前版本
	currentV, err := semver.ParseTolerant(version)
	if err != nil {
		return // 无法解析当前版本，不提示
	}

	// 如果有新版本，则打印提示
	if latest.Version.GT(currentV) {
		// 使用 stderr 输出，避免干扰主命令的 stdout
		fmt.Fprintf(os.Stderr, "\n\n%s\n",
			color.Yellow("提示: Revlay 有一个新版本可用 (%s -> %s)。请运行 'revlay update' 进行更新。", version, latest.Version.String()),
		)
	}
}
