package deployment

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
)

// preflightChecks ensures that the release name is valid.
func (d *LocalDeployer) preflightChecks(releaseName string) error {
	if strings.Contains(releaseName, "..") || strings.Contains(releaseName, "/") || strings.Contains(releaseName, "\\") {
		return fmt.Errorf(i18n.T().ErrorReleaseNotFound, releaseName)
	}
	releasePath := d.config.GetReleasePathByName(releaseName)
	if _, err := os.Stat(releasePath); !os.IsNotExist(err) {
		return fmt.Errorf(i18n.T().DeployFailed, releaseName)
	}
	return nil
}

// setupDirectories creates the necessary directories for deployment.
func (d *LocalDeployer) setupDirectories() error {
	paths := []string{
		d.config.GetReleasesPath(),
		d.config.GetSharedPath(),
		d.config.GetPidsPath(),
		d.config.GetLogsPath(),
	}
	for _, p := range paths {
		fmt.Printf(i18n.T().DeployEnsuringDir+"\n", p)
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", p, err)
		}
	}
	return nil
}

// setupDirectoriesAndRelease creates the directory for a new release and copies the source code.
func (d *LocalDeployer) setupDirectoriesAndRelease(releaseName string, sourceDir string) error {
	if err := d.setupDirectories(); err != nil {
		return err
	}

	fmt.Println(color.Cyan(i18n.T().DeployPopulatingDir))
	releasePath := d.config.GetReleasePathByName(releaseName)
	if sourceDir != "" {
		fmt.Printf(i18n.T().DeployCopyingContent+"\n", sourceDir, releasePath)
		if err := copyDirectory(sourceDir, releasePath); err != nil {
			return fmt.Errorf("failed to copy from source directory %s: %w", sourceDir, err)
		}
	} else {
		if err := os.MkdirAll(releasePath, 0755); err != nil {
			return fmt.Errorf("failed to create release directory %s: %w", releasePath, err)
		}
		fmt.Printf(i18n.T().DeployCreatedEmpty+"\n", releasePath)
		fmt.Println(color.Yellow(i18n.T().DeployEmptyNote))
	}
	return nil
}

// linkSharedPaths creates symlinks for shared paths defined in the config.
func (d *LocalDeployer) linkSharedPaths(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	sharedPath := d.config.GetSharedPath()

	fmt.Println(color.Cyan(i18n.T().DeployLinkingShared))

	// Link shared files
	for _, file := range d.config.Deploy.SharedFiles {
		sourcePath := filepath.Join(sharedPath, file)
		destPath := filepath.Join(releasePath, file)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for shared file link: %w", err)
		}

		// Remove if it exists (e.g., copied from source)
		if _, err := os.Lstat(destPath); err == nil {
			if err := os.Remove(destPath); err != nil {
				return fmt.Errorf("failed to remove existing file at %s: %w", destPath, err)
			}
		}

		fmt.Printf("  -> "+i18n.T().DeployLinking+"\n", file)
		if err := os.Symlink(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to link shared file %s: %w", file, err)
		}
	}

	// Link shared directories
	for _, dir := range d.config.Deploy.SharedDirs {
		sourcePath := filepath.Join(sharedPath, dir)
		destPath := filepath.Join(releasePath, dir)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for shared dir link: %w", err)
		}

		// Remove if it exists
		if _, err := os.Lstat(destPath); err == nil {
			if err := os.RemoveAll(destPath); err != nil {
				return fmt.Errorf("failed to remove existing dir at %s: %w", destPath, err)
			}
		}

		fmt.Printf("  -> "+i18n.T().DeployLinking+"\n", dir)
		if err := os.Symlink(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to link shared directory %s: %w", dir, err)
		}
	}

	return nil
}

// switchSymlink points the 'current' symlink to the specified release.
func (d *LocalDeployer) switchSymlink(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	currentPath := d.config.GetCurrentPath()

	fmt.Printf(i18n.T().DeployPointingSymlink+"\n", releasePath)

	// Create a temporary symlink
	tempLink := currentPath + ".tmp"
	if err := os.Symlink(releasePath, tempLink); err != nil {
		return fmt.Errorf(i18n.T().DeployPointingSymlink, err)
	}

	// Atomically rename the temporary symlink to the final name
	if err := os.Rename(tempLink, currentPath); err != nil {
		return fmt.Errorf(i18n.T().DeployRenameFailed, err)
	}

	return nil
}

// Prune removes old releases, keeping a configured number of recent ones.
func (d *LocalDeployer) Prune() error {
	keep := d.config.App.KeepReleases
	if keep <= 0 {
		return nil // Pruning is disabled
	}

	releases, err := d.ListReleases()
	if err != nil {
		return err
	}

	current, err := d.GetCurrentRelease()
	if err != nil {
		// Log a warning but continue. We might not have a current release yet.
		// In this case, we'll just keep the most recent N releases.
	}

	if len(releases) <= keep {
		return nil // Not enough releases to prune
	}

	// Sort releases chronologically (oldest first)
	sort.Strings(releases)

	toPruneCount := len(releases) - keep
	prunedCount := 0
	for _, releaseName := range releases {
		if prunedCount >= toPruneCount {
			break
		}
		if releaseName == current {
			continue // Never prune the current release
		}
		releasePath := d.config.GetReleasePathByName(releaseName)
		fmt.Printf("  -> "+i18n.T().DeployPruningRelease+"\n", releaseName)
		if err := os.RemoveAll(releasePath); err != nil {
			// Log error but continue trying to prune others
			fmt.Printf("  -> Failed to prune %s: %v\n", releaseName, err)
		}
		prunedCount++
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
