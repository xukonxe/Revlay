package deployment

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/config"
)

// Deployer defines the interface for deployment operations.
type Deployer interface {
	Deploy(releaseName string, sourceDir string) error
	Rollback(releaseName string) error
	ListReleases() ([]string, error)
	GetCurrentRelease() (string, error)
	Prune() error
}

// Release represents a deployment release.
type Release struct {
	Name    string
	Path    string
	Current bool
}

// LocalDeployer handles deployments on the local machine.
type LocalDeployer struct {
	config *config.Config
}

// NewLocalDeployer creates a new deployer for local operations.
func NewLocalDeployer(cfg *config.Config) Deployer {
	return &LocalDeployer{
		config: cfg,
	}
}

// runLocalCommand executes a command on the local machine.
func (d *LocalDeployer) runLocalCommand(name string, arg ...string) (string, error) {
	fmt.Println(color.Cyan("  -> Executing: %s %s", name, strings.Join(arg, " ")))
	cmd := exec.Command(name, arg...)
	cmd.Dir = d.config.GetReleasePathByName("") // Fallback directory
	
	// If a specific release path exists, run the command there.
	// This part needs context, assuming release path is created before running commands inside it.
	// For now, let's keep it simple. A better approach might be to pass the releaseName.

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s\n%s", err, output)
	}
	return string(output), nil
}

// Deploy performs a standard deployment on the local machine.
func (d *LocalDeployer) Deploy(releaseName string, sourceDir string) error {
	// 1. Setup directories
	fmt.Println(color.Cyan("Step 1: Setting up directories..."))
	if err := d.setupDirectories(); err != nil {
		return err
	}

	// 2. Create the release content
	fmt.Println(color.Cyan("Step 2: Populating release directory..."))
	releasePath := d.config.GetReleasePathByName(releaseName)
	if sourceDir != "" {
		// Move content from source directory
		fmt.Printf("  - Moving content from %s\n", sourceDir)
		if err := os.Rename(sourceDir, releasePath); err != nil {
			// Fallback to copy if rename fails (e.g., across different filesystems)
			fmt.Println(color.Yellow("  - Rename failed, falling back to copy..."))
			if _, err := d.runLocalCommand("cp", "-r", sourceDir, releasePath); err != nil {
				return fmt.Errorf("failed to copy from source directory %s: %w", sourceDir, err)
			}
		}

	} else {
		// Create an empty directory
		if err := os.MkdirAll(releasePath, 0755); err != nil {
			return fmt.Errorf("failed to create release directory %s: %w", releasePath, err)
		}
		fmt.Printf("  - Created empty release directory: %s\n", releasePath)
		fmt.Println(color.Yellow("  - Note: No source specified. Use pre_deploy hooks to populate this directory."))
	}

	// 3. Link shared paths
	fmt.Println(color.Cyan("Step 3: Linking shared paths..."))
	if err := d.linkSharedPaths(releaseName); err != nil {
		return err
	}

	// 4. Run pre-deploy hooks
	if len(d.config.Hooks.PreDeploy) > 0 {
		fmt.Println(color.Cyan("Step 4: Running pre-deploy hooks..."))
		for _, hook := range d.config.Hooks.PreDeploy {
			if _, err := d.runLocalCommand("sh", "-c", hook); err != nil {
				return err
			}
		}
	}
	
	// 5. Switch symlink
	fmt.Println(color.Cyan("Step 5: Activating new release..."))
	if err := d.switchSymlink(releaseName); err != nil {
		return err
	}

	// 6. Run restart command
	if d.config.Service.RestartCommand != "" {
		fmt.Println(color.Cyan("Step 6: Restarting service..."))
		if _, err := d.runLocalCommand("sh", "-c", d.config.Service.RestartCommand); err != nil {
			return fmt.Errorf("failed to run restart command: %w", err)
		}
	}

	// 7. Perform health check
	if d.config.Service.HealthCheck != "" {
		fmt.Println(color.Cyan("Step 7: Performing health check..."))
		if err := d.waitForService(d.config.Service.Port); err != nil {
			return fmt.Errorf("health check failed after restart: %w", err)
		}
		fmt.Println(color.Green("  - Health check passed."))
	}

	// 8. Run post-deploy hooks
	if len(d.config.Hooks.PostDeploy) > 0 {
		fmt.Println(color.Cyan("Step 8: Running post-deploy hooks..."))
		for _, hook := range d.config.Hooks.PostDeploy {
			if _, err := d.runLocalCommand("sh", "-c", hook); err != nil {
				return err
			}
		}
	}

	// 9. Prune old releases
	fmt.Println(color.Cyan("Step 9: Pruning old releases..."))
	return d.Prune()
}

// DeployZeroDowntime performs a zero-downtime deployment.
// For now, this will be a simplified version. A full implementation
// would involve health checks and port switching logic.
func (d *LocalDeployer) DeployZeroDowntime(releaseName string, sourceDir string) error {
	fmt.Println(color.Yellow("Warning: Zero-downtime deployment is currently simplified and acts like a standard deploy."))
	return d.Deploy(releaseName, sourceDir)
}

// Rollback reverts to a previous release.
func (d *LocalDeployer) Rollback(releaseName string) error {
	// Verify the release to rollback to actually exists
	releasePath := d.config.GetReleasePathByName(releaseName)
	if _, err := os.Stat(releasePath); os.IsNotExist(err) {
		return fmt.Errorf("cannot roll back: release '%s' does not exist", releaseName)
	}

	fmt.Printf("Rolling back to release %s...\n", releaseName)
	if err := d.switchSymlink(releaseName); err != nil {
		return err
	}
	fmt.Println("Rollback successful.")
	return nil
}

// ListReleases lists all releases on the local filesystem.
func (d *LocalDeployer) ListReleases() ([]string, error) {
	releasesPath := d.config.GetReleasesPath()
	files, err := os.ReadDir(releasesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No releases yet, return empty list
		}
		return nil, err
	}

	var releases []string
	for _, file := range files {
		if file.IsDir() {
			releases = append(releases, file.Name())
		}
	}
	// Sort by name (timestamp)
	sort.Strings(releases)
	return releases, nil
}


// Prune removes old releases according to the keep_releases setting.
func (d *LocalDeployer) Prune() error {
	keep := d.config.App.KeepReleases
	if keep < 0 {
		// keep = -1 means keep all releases
		return nil
	}

	releases, err := d.ListReleases()
	if err != nil {
		return err
	}

	// Sort releases by name, which should correspond to date
	sort.Slice(releases, func(i, j int) bool {
		return releases[i] < releases[j]
	})

	var releasesToRemove []string
	currentReleaseName, _ := d.GetCurrentRelease()
	
	// How many releases we can safely remove
	canRemoveCount := len(releases) - keep

	for _, release := range releases {
		if canRemoveCount <= 0 {
			break
		}
		// Never remove the current release
		if release == currentReleaseName {
			continue
		}
		releasesToRemove = append(releasesToRemove, release)
		canRemoveCount--
	}
	
	for _, release := range releasesToRemove {
		fmt.Printf("Pruning old release: %s\n", release)
		if err := os.RemoveAll(filepath.Join(d.config.GetReleasesPath(), release)); err != nil {
			// Log error but continue trying to prune others
			log.Printf("  - Failed to remove %s: %v\n", release, err)
		}
	}

	return nil
}

// GetCurrentRelease reads the symlink to find the current release name.
func (d *LocalDeployer) GetCurrentRelease() (string, error) {
	currentPath := d.config.GetCurrentPath()
	target, err := os.Readlink(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No current link
		}
		return "", err
	}
	return filepath.Base(target), nil
}

func (d *LocalDeployer) waitForService(port int) error {
	maxRetries := 30
	if d.config.Service.RestartDelay > 0 {
		maxRetries = d.config.Service.RestartDelay
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	healthCheckURL := fmt.Sprintf("http://localhost:%d%s", port, d.config.Service.HealthCheck)

	for i := 0; i < maxRetries; i++ {
		fmt.Printf(color.Yellow("  - Health check attempt #%d for %s... ", i+1, healthCheckURL))
		resp, err := client.Get(healthCheckURL)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			resp.Body.Close()
			fmt.Println(color.Green("OK"))
			return nil // Service is healthy
		}
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Println(color.Red("Failed"))
		time.Sleep(2 * time.Second) // wait before next retry
	}

	return fmt.Errorf("service not responding after %d attempts", maxRetries)
}

func (d *LocalDeployer) setupDirectories() error {
	paths := []string{
		d.config.GetReleasesPath(),
		d.config.GetSharedPath(),
	}
	for _, path := range paths {
		fmt.Printf("  - Ensuring directory exists: %s\n", path)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (d *LocalDeployer) linkSharedPaths(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	sharedPath := d.config.GetSharedPath()

	// Ensure the base shared directory exists, so we don't fail if it's empty
	if err := os.MkdirAll(sharedPath, 0755); err != nil {
		return fmt.Errorf("failed to create base shared directory: %w", err)
	}

	items, err := os.ReadDir(sharedPath)
	if err != nil {
		return fmt.Errorf("failed to read shared directory: %w", err)
	}

	for _, item := range items {
		target := filepath.Join(sharedPath, item.Name())
		link := filepath.Join(releasePath, item.Name())
		fmt.Printf("  - Linking: %s -> %s\n", link, target)
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", item.Name(), err)
		}
	}

	return nil
}

func (d *LocalDeployer) switchSymlink(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	currentPath := d.config.GetCurrentPath()

	fmt.Printf("  - Pointing 'current' symlink to: %s\n", releasePath)

	tempLink := currentPath + "_tmp"
	if err := os.Symlink(releasePath, tempLink); err != nil {
		return err
	}
	if err := os.Rename(tempLink, currentPath); err != nil {
		return err
	}
	return nil
}

// GenerateReleaseTimestamp creates a timestamp string for a release.
func GenerateReleaseTimestamp() string {
	return time.Now().Format("20060102150405")
}