package deployment

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
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

// DeployZeroDowntime performs zero-downtime deployment using blue-green strategy
func (d *Deployer) DeployZeroDowntime(releaseName string) error {
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

	// Determine ports for blue-green deployment
	currentPort, newPort, err := d.determinePorts()
	if err != nil {
		return fmt.Errorf("failed to determine ports: %w", err)
	}

	// Start new service on alternative port
	if err := d.startServiceOnPort(releaseName, newPort); err != nil {
		return fmt.Errorf("failed to start service on port %d: %w", newPort, err)
	}

	// Wait for service to be ready
	if err := d.waitForService(newPort); err != nil {
		// Cleanup failed service
		d.stopServiceOnPort(newPort)
		return fmt.Errorf("service failed to start: %w", err)
	}

	// Switch current symlink to new release
	if err := d.SwitchCurrent(releaseName); err != nil {
		// Cleanup failed deployment
		d.stopServiceOnPort(newPort)
		return fmt.Errorf("failed to switch current: %w", err)
	}

	// Update load balancer or reverse proxy to point to new port
	if err := d.switchTrafficToPort(newPort); err != nil {
		// Try to rollback
		d.SwitchCurrent(d.getPreviousRelease())
		d.stopServiceOnPort(newPort)
		return fmt.Errorf("failed to switch traffic: %w", err)
	}

	// Gracefully shutdown old service
	if currentPort != 0 {
		if err := d.gracefulShutdownService(currentPort); err != nil {
			// Log error but don't fail deployment
			fmt.Printf("Warning: failed to gracefully shutdown old service on port %d: %v\n", currentPort, err)
		}
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

// determinePorts determines the current and new ports for blue-green deployment
func (d *Deployer) determinePorts() (int, int, error) {
	// Check if there's a current release running
	currentRelease, err := d.GetCurrentRelease()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get current release: %w", err)
	}

	if currentRelease == "" {
		// No current release, use primary port
		return 0, d.config.Service.Port, nil
	}

	// Determine which port is currently in use
	currentPort, err := d.getCurrentServicePort()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to determine current service port: %w", err)
	}

	// Switch to alternative port
	if currentPort == d.config.Service.Port {
		return currentPort, d.config.Service.AltPort, nil
	}
	return currentPort, d.config.Service.Port, nil
}

// getCurrentServicePort determines which port the current service is running on
func (d *Deployer) getCurrentServicePort() (int, error) {
	// Check if service is running on primary port
	if d.isServiceRunning(d.config.Service.Port) {
		return d.config.Service.Port, nil
	}
	
	// Check if service is running on alternative port
	if d.isServiceRunning(d.config.Service.AltPort) {
		return d.config.Service.AltPort, nil
	}
	
	// No service running, default to primary port
	return d.config.Service.Port, nil
}

// isServiceRunning checks if a service is running on the specified port
func (d *Deployer) isServiceRunning(port int) bool {
	cmd := fmt.Sprintf("netstat -tlnp | grep ':%d ' | wc -l", port)
	output, err := d.ssh.RunCommand(cmd)
	if err != nil {
		return false
	}
	
	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return false
	}
	
	return count > 0
}

// startServiceOnPort starts the service on the specified port
func (d *Deployer) startServiceOnPort(releaseName string, port int) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	
	// Set environment variables for the service
	env := make(map[string]string)
	for k, v := range d.config.Deploy.Environment {
		env[k] = v
	}
	env["PORT"] = fmt.Sprintf("%d", port)
	env["RELEASE_PATH"] = releasePath
	
	// Build service start command
	cmd := d.buildServiceCommand(d.config.Service.Command, port, releasePath)
	
	// Start the service
	if _, err := d.ssh.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	
	return nil
}

// buildServiceCommand builds the service command with port and path substitution
func (d *Deployer) buildServiceCommand(baseCommand string, port int, releasePath string) string {
	cmd := baseCommand
	cmd = strings.ReplaceAll(cmd, "${PORT}", fmt.Sprintf("%d", port))
	cmd = strings.ReplaceAll(cmd, "${RELEASE_PATH}", releasePath)
	cmd = strings.ReplaceAll(cmd, "${CURRENT_PATH}", d.config.GetCurrentPath())
	cmd = strings.ReplaceAll(cmd, "${SHARED_PATH}", d.config.GetSharedPath())
	return cmd
}

// waitForService waits for the service to be ready on the specified port
func (d *Deployer) waitForService(port int) error {
	maxRetries := 30
	retryDelay := time.Duration(d.config.Service.RestartDelay) * time.Second
	
	for i := 0; i < maxRetries; i++ {
		if d.isServiceHealthy(port) {
			return nil
		}
		
		time.Sleep(retryDelay)
	}
	
	return fmt.Errorf("service failed to become healthy after %d retries", maxRetries)
}

// isServiceHealthy checks if the service is healthy on the specified port
func (d *Deployer) isServiceHealthy(port int) bool {
	if d.config.Service.HealthCheck == "" {
		// No health check configured, just check if port is listening
		return d.isServiceRunning(port)
	}
	
	// Perform HTTP health check
	url := fmt.Sprintf("http://localhost:%d%s", port, d.config.Service.HealthCheck)
	cmd := fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' %s", url)
	
	output, err := d.ssh.RunCommand(cmd)
	if err != nil {
		return false
	}
	
	httpCode, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return false
	}
	
	return httpCode >= 200 && httpCode < 400
}

// switchTrafficToPort switches the load balancer or reverse proxy to the new port
func (d *Deployer) switchTrafficToPort(port int) error {
	// This is a placeholder for load balancer/reverse proxy switching logic
	// In a real implementation, this would update nginx, haproxy, etc.
	
	// For now, we'll use a simple approach with a port file
	portFile := filepath.Join(d.config.Deploy.Path, "current_port")
	cmd := fmt.Sprintf("echo %d > %s", port, portFile)
	
	if _, err := d.ssh.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to update port file: %w", err)
	}
	
	return nil
}

// gracefulShutdownService gracefully shuts down the service on the specified port
func (d *Deployer) gracefulShutdownService(port int) error {
	timeout := d.config.Service.GracefulTimeout
	if timeout <= 0 {
		timeout = 30 // Default 30 seconds
	}
	
	// Send graceful shutdown signal
	cmd := fmt.Sprintf("pkill -TERM -f 'port.*%d' || true", port)
	if _, err := d.ssh.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to send graceful shutdown signal: %w", err)
	}
	
	// Wait for graceful shutdown
	time.Sleep(time.Duration(timeout) * time.Second)
	
	// Force kill if still running
	if d.isServiceRunning(port) {
		cmd = fmt.Sprintf("pkill -KILL -f 'port.*%d' || true", port)
		if _, err := d.ssh.RunCommand(cmd); err != nil {
			return fmt.Errorf("failed to force kill service: %w", err)
		}
	}
	
	return nil
}

// stopServiceOnPort stops the service on the specified port
func (d *Deployer) stopServiceOnPort(port int) error {
	cmd := fmt.Sprintf("pkill -KILL -f 'port.*%d' || true", port)
	if _, err := d.ssh.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to stop service on port %d: %w", port, err)
	}
	return nil
}

// getPreviousRelease returns the previous release name
func (d *Deployer) getPreviousRelease() string {
	releases, err := d.ListReleases()
	if err != nil || len(releases) < 2 {
		return ""
	}
	
	// Find current release and return the previous one
	for i, release := range releases {
		if release.Current && i+1 < len(releases) {
			return releases[i+1].Name
		}
	}
	
	return ""
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