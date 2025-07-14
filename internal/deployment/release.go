package deployment

import (
	"fmt"
	"os"
	"path/filepath"
)

// ListReleases lists all available releases.
func (d *LocalDeployer) ListReleases() ([]string, error) {
	releasesPath := d.config.GetReleasesPath()
	files, err := os.ReadDir(releasesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No releases directory yet, so no releases.
		}
		return nil, fmt.Errorf("could not list releases: %w", err)
	}

	var releases []string
	for _, file := range files {
		if file.IsDir() {
			releases = append(releases, file.Name())
		}
	}
	return releases, nil
}

// GetCurrentRelease finds the release the 'current' symlink points to.
func (d *LocalDeployer) GetCurrentRelease() (string, error) {
	currentPath := d.config.GetCurrentPath()
	target, err := os.Readlink(currentPath)
	if err != nil {
		return "", fmt.Errorf("could not read current symlink: %w", err)
	}
	return filepath.Base(target), nil
}
