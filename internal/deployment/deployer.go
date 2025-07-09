package deployment

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/ssh"
)

// Deployer handles deployment operations
type Deployer struct {
	config *config.Config
	ssh    *ssh.Client
}

// Release represents a deployment release
type Release struct {
	Name      string
	Path      string
	Timestamp time.Time
	Current   bool
}

// NewDeployer creates a new deployer instance
func NewDeployer(cfg *config.Config, sshClient *ssh.Client) *Deployer {
	return &Deployer{
		config: cfg,
		ssh:    sshClient,
	}
}

// InitializeDirectories sets up the basic directory structure on the remote server
func (d *Deployer) InitializeDirectories() error {
	// Create main deployment directory
	if err := d.ssh.CreateDir(d.config.Deploy.Path); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	// Create releases directory
	if err := d.ssh.CreateDir(d.config.GetReleasesPath()); err != nil {
		return fmt.Errorf("failed to create releases directory: %w", err)
	}

	// Create shared directory
	if err := d.ssh.CreateDir(d.config.GetSharedPath()); err != nil {
		return fmt.Errorf("failed to create shared directory: %w", err)
	}

	// Create shared subdirectories
	for _, sharedPath := range d.config.Deploy.SharedPaths {
		fullPath := filepath.Join(d.config.GetSharedPath(), sharedPath)
		if err := d.ssh.CreateDir(filepath.Dir(fullPath)); err != nil {
			return fmt.Errorf("failed to create shared path %s: %w", sharedPath, err)
		}
	}

	return nil
}

// CreateRelease creates a new release directory
func (d *Deployer) CreateRelease(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	
	// Create release directory
	if err := d.ssh.CreateDir(releasePath); err != nil {
		return fmt.Errorf("failed to create release directory: %w", err)
	}

	return nil
}

// LinkSharedPaths creates symbolic links for shared paths
func (d *Deployer) LinkSharedPaths(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	
	for _, sharedPath := range d.config.Deploy.SharedPaths {
		targetPath := filepath.Join(d.config.GetSharedPath(), sharedPath)
		linkPath := filepath.Join(releasePath, sharedPath)
		
		// Create parent directory if it doesn't exist
		if err := d.ssh.CreateDir(filepath.Dir(linkPath)); err != nil {
			return fmt.Errorf("failed to create parent directory for shared path %s: %w", sharedPath, err)
		}
		
		// Create symbolic link
		if err := d.ssh.CreateSymlink(targetPath, linkPath); err != nil {
			return fmt.Errorf("failed to create symlink for shared path %s: %w", sharedPath, err)
		}
	}

	return nil
}

// SwitchCurrent switches the current symlink to point to the new release
func (d *Deployer) SwitchCurrent(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	currentPath := d.config.GetCurrentPath()
	
	// Create symbolic link (this is atomic)
	if err := d.ssh.CreateSymlink(releasePath, currentPath); err != nil {
		return fmt.Errorf("failed to switch current symlink: %w", err)
	}

	return nil
}

// GetCurrentRelease returns the name of the current release
func (d *Deployer) GetCurrentRelease() (string, error) {
	currentPath := d.config.GetCurrentPath()
	
	// Check if current symlink exists
	exists, err := d.ssh.FileExists(currentPath)
	if err != nil {
		return "", fmt.Errorf("failed to check current symlink: %w", err)
	}
	
	if !exists {
		return "", nil
	}

	// Read symlink target
	target, err := d.ssh.ReadSymlink(currentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read current symlink: %w", err)
	}

	// Extract release name from path
	return filepath.Base(target), nil
}

// ListReleases returns a list of all releases
func (d *Deployer) ListReleases() ([]Release, error) {
	files, err := d.ssh.ListFiles(d.config.GetReleasesPath())
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	var releases []Release
	currentRelease, _ := d.GetCurrentRelease()

	for _, file := range files {
		release := Release{
			Name:    file,
			Path:    d.config.GetReleasePathByName(file),
			Current: file == currentRelease,
		}
		
		// Try to parse timestamp from release name
		if timestamp, err := time.Parse("20060102-150405", file); err == nil {
			release.Timestamp = timestamp
		}
		
		releases = append(releases, release)
	}

	// Sort releases by timestamp (newest first)
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Timestamp.After(releases[j].Timestamp)
	})

	return releases, nil
}

// CleanupOldReleases removes old releases beyond the keep limit
func (d *Deployer) CleanupOldReleases() error {
	releases, err := d.ListReleases()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Keep only the configured number of releases
	if len(releases) <= d.config.App.KeepReleases {
		return nil
	}

	// Get current release to avoid deleting it
	currentRelease, _ := d.GetCurrentRelease()

	// Remove old releases
	for i := d.config.App.KeepReleases; i < len(releases); i++ {
		release := releases[i]
		
		// Don't delete current release
		if release.Name == currentRelease {
			continue
		}

		if err := d.ssh.RemoveDir(release.Path); err != nil {
			return fmt.Errorf("failed to remove old release %s: %w", release.Name, err)
		}
	}

	return nil
}

// RunHooks executes deployment hooks
func (d *Deployer) RunHooks(hookType string, releaseName string) error {
	var hooks []string
	
	switch hookType {
	case "pre_deploy":
		hooks = d.config.Hooks.PreDeploy
	case "post_deploy":
		hooks = d.config.Hooks.PostDeploy
	case "pre_rollback":
		hooks = d.config.Hooks.PreRollback
	case "post_rollback":
		hooks = d.config.Hooks.PostRollback
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	for _, hook := range hooks {
		// Replace placeholders in hook commands
		hook = strings.ReplaceAll(hook, "${RELEASE_PATH}", d.config.GetReleasePathByName(releaseName))
		hook = strings.ReplaceAll(hook, "${CURRENT_PATH}", d.config.GetCurrentPath())
		hook = strings.ReplaceAll(hook, "${SHARED_PATH}", d.config.GetSharedPath())
		
		// Execute hook
		if _, err := d.ssh.RunCommand(hook); err != nil {
			return fmt.Errorf("failed to execute %s hook '%s': %w", hookType, hook, err)
		}
	}

	return nil
}

// Deploy performs a complete deployment
func (d *Deployer) Deploy(releaseName string) error {
	// Run pre-deploy hooks
	if err := d.RunHooks("pre_deploy", releaseName); err != nil {
		return fmt.Errorf("pre-deploy hooks failed: %w", err)
	}

	// Initialize directories if needed
	if err := d.InitializeDirectories(); err != nil {
		return fmt.Errorf("failed to initialize directories: %w", err)
	}

	// Create release directory
	if err := d.CreateRelease(releaseName); err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	// Link shared paths
	if err := d.LinkSharedPaths(releaseName); err != nil {
		return fmt.Errorf("failed to link shared paths: %w", err)
	}

	// Switch current symlink
	if err := d.SwitchCurrent(releaseName); err != nil {
		return fmt.Errorf("failed to switch current: %w", err)
	}

	// Run post-deploy hooks
	if err := d.RunHooks("post_deploy", releaseName); err != nil {
		return fmt.Errorf("post-deploy hooks failed: %w", err)
	}

	// Cleanup old releases
	if err := d.CleanupOldReleases(); err != nil {
		return fmt.Errorf("failed to cleanup old releases: %w", err)
	}

	return nil
}

// Rollback rolls back to a previous release
func (d *Deployer) Rollback(releaseName string) error {
	// Check if release exists
	exists, err := d.ssh.DirExists(d.config.GetReleasePathByName(releaseName))
	if err != nil {
		return fmt.Errorf("failed to check release existence: %w", err)
	}
	
	if !exists {
		return fmt.Errorf("release %s does not exist", releaseName)
	}

	// Run pre-rollback hooks
	if err := d.RunHooks("pre_rollback", releaseName); err != nil {
		return fmt.Errorf("pre-rollback hooks failed: %w", err)
	}

	// Switch current symlink
	if err := d.SwitchCurrent(releaseName); err != nil {
		return fmt.Errorf("failed to switch current: %w", err)
	}

	// Run post-rollback hooks
	if err := d.RunHooks("post_rollback", releaseName); err != nil {
		return fmt.Errorf("post-rollback hooks failed: %w", err)
	}

	return nil
}

// GenerateReleaseTimestamp generates a timestamp-based release name
func GenerateReleaseTimestamp() string {
	return time.Now().Format("20060102-150405")
}

// GenerateUniqueReleaseTimestamp generates a unique timestamp-based release name
func GenerateUniqueReleaseTimestamp() string {
	now := time.Now()
	return fmt.Sprintf("%s-%d", now.Format("20060102-150405"), now.UnixNano()%1000000)
}