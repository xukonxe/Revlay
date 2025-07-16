package deployment

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xukonxe/revlay/internal/i18n"
)

// preflightChecks ensures that the release name is valid.
func (d *LocalDeployer) preflightChecks(releaseName string, logger *stepLogger) error {
	if logger != nil {
		logger.SystemLog(fmt.Sprintf("检查版本名称有效性: %s", releaseName))
	}

	if strings.Contains(releaseName, "..") || strings.Contains(releaseName, "/") || strings.Contains(releaseName, "\\") {
		return fmt.Errorf(i18n.T().ErrorReleaseNotFound, releaseName)
	}

	releasePath := d.config.GetReleasePathByName(releaseName)
	if logger != nil {
		logger.SystemLog(fmt.Sprintf("检查版本目录是否已存在: %s", releasePath))
	}

	if _, err := os.Stat(releasePath); !os.IsNotExist(err) {
		return fmt.Errorf(i18n.T().DeployFailed, releaseName)
	}

	if logger != nil {
		logger.SystemLog("预检完成: 版本名称有效且不存在冲突")
	}
	return nil
}

// setupDirectories creates the necessary directories for deployment.
func (d *LocalDeployer) setupDirectories(logger *stepLogger) error {
	paths := []string{
		d.config.GetReleasesPath(),
		d.config.GetSharedPath(),
		d.config.GetPidsPath(),
		d.config.GetLogsPath(),
	}
	for _, p := range paths {
		if logger != nil {
			logger.SystemLog(fmt.Sprintf("确保目录存在: %s", p))
		}
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", p, err)
		}
	}
	return nil
}

// setupDirectoriesAndRelease creates the directory for a new release and copies the source code.
func (d *LocalDeployer) setupDirectoriesAndRelease(releaseName string, sourceDir string, logger *stepLogger) error {
	// 创建目录结构
	if logger != nil {
		logger.SystemLog("开始创建必要的目录结构...")
	}

	if err := d.setupDirectories(logger); err != nil {
		return err
	}

	// 创建新版本目录
	releasePath := d.config.GetReleasePathByName(releaseName)
	if logger != nil {
		logger.SystemLog(fmt.Sprintf("准备创建版本目录: %s", releasePath))
	}

	// 如果提供了源目录，则复制内容
	if sourceDir != "" {
		if logger != nil {
			logger.SystemLog(fmt.Sprintf("从源目录复制内容: %s -> %s", sourceDir, releasePath))
		}
		if err := copyDirectory(sourceDir, releasePath); err != nil {
			return fmt.Errorf("failed to copy from source directory %s: %w", sourceDir, err)
		}
		if logger != nil {
			logger.SystemLog("源目录内容复制完成")
		}
	} else {
		// 否则创建一个空目录
		if logger != nil {
			logger.SystemLog(fmt.Sprintf("创建空版本目录: %s", releasePath))
		}
		if err := os.MkdirAll(releasePath, 0755); err != nil {
			return fmt.Errorf("failed to create release directory %s: %w", releasePath, err)
		}
		if logger != nil {
			logger.SystemLog("空版本目录创建完成")
		}
	}

	return nil
}

// linkSharedPaths creates symlinks for shared paths defined in the config.
func (d *LocalDeployer) linkSharedPaths(releaseName string, logger *stepLogger) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	sharedPath := d.config.GetSharedPath()

	if logger != nil {
		logger.SystemLog(fmt.Sprintf("开始创建共享路径链接，从 %s 到 %s", sharedPath, releasePath))
	}

	// Link shared files
	for _, file := range d.config.Deploy.SharedFiles {
		sourcePath := filepath.Join(sharedPath, file)
		destPath := filepath.Join(releasePath, file)

		if logger != nil {
			logger.SystemLog(fmt.Sprintf("准备链接共享文件: %s", file))
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for shared file link: %w", err)
		}

		// Remove if it exists (e.g., copied from source)
		if _, err := os.Lstat(destPath); err == nil {
			if logger != nil {
				logger.SystemLog(fmt.Sprintf("删除已存在的文件: %s", destPath))
			}
			if err := os.Remove(destPath); err != nil {
				return fmt.Errorf("failed to remove existing file at %s: %w", destPath, err)
			}
		}

		if logger != nil {
			logger.SystemLog(fmt.Sprintf("创建符号链接: %s -> %s", sourcePath, destPath))
		}
		if err := os.Symlink(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to link shared file %s: %w", file, err)
		}
	}

	// Link shared directories
	for _, dir := range d.config.Deploy.SharedDirs {
		sourcePath := filepath.Join(sharedPath, dir)
		destPath := filepath.Join(releasePath, dir)

		if logger != nil {
			logger.SystemLog(fmt.Sprintf("准备链接共享目录: %s", dir))
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for shared dir link: %w", err)
		}

		// Remove if it exists
		if _, err := os.Lstat(destPath); err == nil {
			if logger != nil {
				logger.SystemLog(fmt.Sprintf("删除已存在的目录: %s", destPath))
			}
			if err := os.RemoveAll(destPath); err != nil {
				return fmt.Errorf("failed to remove existing dir at %s: %w", destPath, err)
			}
		}

		if logger != nil {
			logger.SystemLog(fmt.Sprintf("创建符号链接: %s -> %s", sourcePath, destPath))
		}
		if err := os.Symlink(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to link shared directory %s: %w", dir, err)
		}
	}

	if logger != nil {
		logger.SystemLog("共享路径链接创建完成")
	}
	return nil
}

// switchSymlink points the 'current' symlink to the specified release.
func (d *LocalDeployer) switchSymlink(releaseName string, logger *stepLogger) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	currentPath := d.config.GetCurrentPath()

	if logger != nil {
		logger.SystemLog(fmt.Sprintf("准备将 'current' 符号链接指向新版本: %s", releaseName))
	}

	// Create a temporary symlink
	tempLink := currentPath + ".tmp"
	if logger != nil {
		logger.SystemLog(fmt.Sprintf("创建临时符号链接: %s -> %s", tempLink, releasePath))
	}
	if err := os.Symlink(releasePath, tempLink); err != nil {
		return fmt.Errorf(i18n.T().DeployPointingSymlink, err)
	}

	// Atomically rename the temporary symlink to the final name
	if logger != nil {
		logger.SystemLog(fmt.Sprintf("原子重命名临时链接: %s -> %s", tempLink, currentPath))
	}
	if err := os.Rename(tempLink, currentPath); err != nil {
		return fmt.Errorf(i18n.T().DeployRenameFailed, err)
	}

	if logger != nil {
		logger.SystemLog(fmt.Sprintf("'current' 符号链接已成功指向新版本: %s", releaseName))
	}
	return nil
}

// Prune removes old releases, keeping a configured number of recent ones.
func (d *LocalDeployer) Prune(logger *stepLogger) error {
	keep := d.config.App.KeepReleases
	if keep <= 0 {
		if logger != nil {
			logger.SystemLog("清理已禁用 (keep_releases <= 0)")
		}
		return nil // Pruning is disabled
	}

	releases, err := d.ListReleases()
	if err != nil {
		return err
	}

	if len(releases) <= keep {
		if logger != nil {
			logger.SystemLog(fmt.Sprintf("无需清理，当前版本数量 (%d) 不超过保留数量 (%d)", len(releases), keep))
		}
		return nil // Not enough releases to prune
	}

	current, _ := d.GetCurrentRelease()
	// Sort releases chronologically (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(releases)))

	if logger != nil {
		logger.SystemLog(fmt.Sprintf("找到 %d 个版本，将保留最新的 %d 个版本", len(releases), keep))
		if current != "" {
			logger.SystemLog(fmt.Sprintf("当前使用的版本是: %s (将被保留)", current))
		}
	}

	releasesToKeep := make(map[string]struct{})

	// Add the current release to the keep list
	if current != "" {
		releasesToKeep[current] = struct{}{}
	}

	// Add the newer releases to the keep list, ensuring we don't exceed 'keep' total
	for _, release := range releases {
		if len(releasesToKeep) >= keep {
			break
		}
		releasesToKeep[release] = struct{}{}
		if logger != nil && release != current {
			logger.SystemLog(fmt.Sprintf("将保留版本: %s", release))
		}
	}

	// Prune releases that are not in our keep list
	for _, releaseName := range releases {
		if _, ok := releasesToKeep[releaseName]; ok {
			continue
		}

		releasePath := d.config.GetReleasePathByName(releaseName)
		if logger != nil {
			logger.SystemLog(fmt.Sprintf("正在清理版本: %s", releaseName))
		}

		// 1. Remove release directory
		if err := os.RemoveAll(releasePath); err != nil {
			// Log error but continue trying to prune others
			if logger != nil {
				logger.SystemLog(fmt.Sprintf("清理版本目录失败: %s - %v", releaseName, err))
			}
			return fmt.Errorf("failed to prune %s: %v", releaseName, err)
		}

		// 2. Remove log files associated with this release
		stdoutLogPath := d.resolvePath(d.config.Service.StdoutLog, releaseName)
		stderrLogPath := d.resolvePath(d.config.Service.StderrLog, releaseName)

		// Remove stdout log file if it exists and is not the same as stderr log
		if _, err := os.Stat(stdoutLogPath); err == nil {
			if logger != nil {
				logger.SystemLog(fmt.Sprintf("正在删除日志文件: %s", stdoutLogPath))
			}
			if err := os.Remove(stdoutLogPath); err != nil {
				if logger != nil {
					logger.SystemLog(fmt.Sprintf("删除日志文件失败: %s - %v", stdoutLogPath, err))
				}
				return fmt.Errorf("failed to remove log file %s: %v", stdoutLogPath, err)
			}
		}

		// Remove stderr log file if it exists and is different from stdout log
		if stderrLogPath != stdoutLogPath {
			if _, err := os.Stat(stderrLogPath); err == nil {
				if logger != nil {
					logger.SystemLog(fmt.Sprintf("正在删除日志文件: %s", stderrLogPath))
				}
				if err := os.Remove(stderrLogPath); err != nil {
					if logger != nil {
						logger.SystemLog(fmt.Sprintf("删除日志文件失败: %s - %v", stderrLogPath, err))
					}
					return fmt.Errorf("failed to remove log file %s: %v", stderrLogPath, err)
				}
			}
		}
	}

	return nil
}

// resolvePath resolves a path template with the release name and makes it absolute.
func (d *LocalDeployer) resolvePath(pathTemplate string, releaseName string) string {
	resolved, _ := d.resolveTemplate(pathTemplate, releaseName)
	// If the path is already absolute, don't join it with the root path.
	if filepath.IsAbs(resolved) {
		return resolved
	}
	return filepath.Join(d.config.RootPath, resolved)
}

// copyDirectory copies a directory from src to dest.
func copyDirectory(src, dest string) error {
	// Create the destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Skip symlinks for now to avoid complexity.
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		return copyRegularFile(path, destPath, info.Mode())
	})
}

// copyRegularFile copies a single regular file.
func copyRegularFile(src, dest string, mode os.FileMode) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
