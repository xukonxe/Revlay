package deployment

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofrs/flock"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/i18n"
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
		return "", fmt.Errorf(i18n.T().DeployCmdExecFailed, err, output)
	}
	return string(output), nil
}

// Deploy dispatches the deployment to the correct strategy based on config.
func (d *LocalDeployer) Deploy(releaseName string, sourceDir string) error {
	lockPath := filepath.Join(d.config.RootPath, "revlay.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf(i18n.T().DeployLockError, err)
	}
	if !locked {
		return fmt.Errorf(i18n.T().DeployAlreadyInProgress)
	}
	defer fileLock.Unlock()

	switch d.config.Deploy.Mode {
	case config.ZeroDowntimeMode:
		return d.deployZeroDowntime(releaseName, sourceDir)
	case config.ShortDowntimeMode:
		return d.deployShortDowntime(releaseName, sourceDir)
	default:
		log.Printf("Unknown deployment mode '%s', falling back to short_downtime.", d.config.Deploy.Mode)
		return d.deployShortDowntime(releaseName, sourceDir)
	}
}

func (d *LocalDeployer) deployShortDowntime(releaseName string, sourceDir string) error {
	fmt.Println(color.Cyan(i18n.T().DeployExecShortDowntime))
	log := newStepLogger()

	// Step 1: Pre-flight checks
	log.Print(i18n.T().DeployPreflightChecks)
	if err := d.preflightChecks(releaseName); err != nil {
		return err
	}

	// Step 2: Setup directories
	log.Print(i18n.T().DeploySetupDirs)
	if err := d.setupDirectoriesAndRelease(releaseName, sourceDir); err != nil {
		return err
	}
	if err := d.linkSharedPaths(releaseName); err != nil {
		return err
	}

	// Step 3: Stop the current service
	log.Print(i18n.T().DeployStoppingService)
	if err := d.stopService(); err != nil {
		log.Warn(fmt.Sprintf(i18n.T().DeployStopServiceFailed, err))
	}

	// Step 4: Activate new release
	log.Print(i18n.T().DeployActivating)
	if err := d.switchSymlink(releaseName); err != nil {
		return err
	}

	// Step 5: Start the new service
	log.Print(i18n.T().DeployStartingService)
	if err := d.startService(releaseName); err != nil {
		return fmt.Errorf(i18n.T().DeployStartServiceFailed, err)
	}

	// Step 6: Perform health check
	log.Print(i18n.T().DeployHealthCheck)
	if err := d.performHealthCheck(d.config.Service.Port); err != nil {
		return err
	}

	// Step 7: Prune old releases
	log.Print(i18n.T().DeployPruning)
	return d.Prune()
}

func (d *LocalDeployer) deployZeroDowntime(releaseName string, sourceDir string) error {
	fmt.Println(color.Cyan(i18n.T().DeployExecZeroDowntime))

	// Step 1: Setup
	fmt.Println(color.Cyan(i18n.T().DeployStep, 1, i18n.T().DeploySetupDirs))
	if err := d.setupDirectoriesAndRelease(releaseName, sourceDir); err != nil {
		return err
	}
	if err := d.linkSharedPaths(releaseName); err != nil {
		return err
	}

	// Step 2: Determine ports
	fmt.Println(color.Cyan(i18n.T().DeployStep, 2, i18n.T().DeployDeterminePorts))
	oldPort, err := d.getCurrentPortFromState()
	if err != nil {
		log.Print(color.Yellow(fmt.Sprintf("Warning: could not determine current port: %v. Defaulting to main port.", err)))
		oldPort = d.config.Service.Port
	}
	newPort := d.config.Service.AltPort
	if oldPort == d.config.Service.AltPort {
		newPort = d.config.Service.Port
	}
	fmt.Printf(i18n.T().DeployCurrentPortInfo+"\n", oldPort)
	fmt.Printf(i18n.T().DeployNewPortInfo+"\n", newPort)

	// Step 3: Start the new version
	fmt.Println(color.Cyan(i18n.T().DeployStep, 3, fmt.Sprintf(i18n.T().DeployStartNewRelease, newPort)))
	var newReleaseCmd *exec.Cmd
	var processDone <-chan error
	if d.config.Service.StartCommand != "" {
		env := map[string]string{"PORT": fmt.Sprintf("%d", newPort)}
		var err error
		newReleaseCmd, processDone, err = d.runCommandAttachedAsyncWithStreaming(releaseName, d.config.Service.StartCommand, env)
		if err != nil {
			return fmt.Errorf("failed to run start command for new release: %w", err)
		}
	}

	// Step 4: Perform health check while monitoring the process
	fmt.Println(color.Cyan(i18n.T().DeployStep, 4, fmt.Sprintf(i18n.T().DeployHealthCheckOnPort, newPort)))

	healthCheckDone := make(chan error, 1)
	go func() {
		healthCheckDone <- d.performHealthCheck(newPort)
	}()

	select {
	case err := <-processDone:
		// Process exited before health check could complete. This is a failure.
		if err == nil {
			return fmt.Errorf(i18n.T().DeployErrProcExitedEarly)
		}
		return fmt.Errorf(i18n.T().DeployErrProcExitedEarlyWithError, err)
	case err := <-healthCheckDone:
		if err != nil {
			// Health check failed. The process might still be running.
			// The original logic to kill the process is implicitly handled now,
			// as the `processDone` channel will receive an error when we kill it.
			// But it's better to be explicit.
			if newReleaseCmd != nil && newReleaseCmd.Process != nil {
				log.Printf("Health check failed. Killing new process (PID: %d)...", newReleaseCmd.Process.Pid)
				if errKill := newReleaseCmd.Process.Kill(); errKill != nil {
					log.Print(color.Yellow(fmt.Sprintf("Warning: failed to kill process %d: %v", newReleaseCmd.Process.Pid, errKill)))
				}
			}
			return fmt.Errorf("health check failed for new release on port %d: %w", newPort, err)
		}
		// Health check passed, deployment can continue.
		fmt.Println(color.Green("  -> " + i18n.T().DeployHealthPassed))
	}

	// Step 5: Switch proxy traffic
	fmt.Println(color.Green(i18n.T().DeployStep, 5, fmt.Sprintf(i18n.T().DeploySwitchProxy, newPort)))
	if err := d.writeStateFile(newPort); err != nil {
		if newReleaseCmd != nil && newReleaseCmd.Process != nil {
			log.Printf("Failed to write active port to state file. Killing new process (PID: %d)...", newReleaseCmd.Process.Pid)
			if errKill := newReleaseCmd.Process.Kill(); errKill != nil {
				log.Print(color.Yellow(fmt.Sprintf("Warning: failed to kill process %d: %v", newReleaseCmd.Process.Pid, errKill)))
			}
		}
		return fmt.Errorf("failed to write active port to state file: %w", err)
	}

	// Step 6: Activate new release symlink
	fmt.Println(color.Cyan(i18n.T().DeployStep, 6, i18n.T().DeployActivateSymlink))
	if err := d.switchSymlink(releaseName); err != nil {
		return err
	}

	// Step 7: Stop old service
	gracePeriod := time.Duration(d.config.Service.GracefulTimeout) * time.Second
	fmt.Println(color.Cyan(i18n.T().DeployStep, 7, fmt.Sprintf(i18n.T().DeployStopOldService, oldPort, gracePeriod)))
	if gracePeriod > 0 {
		time.Sleep(gracePeriod)
	}
	// TODO: refactor stopService for zero-downtime
	// if err := d.stopServiceByPort(oldPort); err != nil { ... }

	// Step 8: Prune old releases
	fmt.Println(color.Cyan(i18n.T().DeployStep, 8, i18n.T().DeployPruning))
	return d.Prune()
}

// DeployZeroDowntime performs a zero-downtime deployment.
// For now, this will be a simplified version. A full implementation
// would involve health checks and port switching logic.
func (d *LocalDeployer) DeployZeroDowntime(releaseName string, sourceDir string) error {
	fmt.Println(color.Yellow(i18n.T().DeployZeroDowntimeWarning))
	return d.Deploy(releaseName, sourceDir)
}

// Rollback reverts to a previous release.
func (d *LocalDeployer) Rollback(releaseName string) error {
	// Verify the release to rollback to actually exists
	releasePath := d.config.GetReleasePathByName(releaseName)
	if _, err := os.Stat(releasePath); os.IsNotExist(err) {
		return fmt.Errorf("cannot roll back: release '%s' does not exist", releaseName)
	}

	fmt.Printf(i18n.T().DeployRollbackStart+"\n", releaseName)
	if err := d.switchSymlink(releaseName); err != nil {
		return err
	}
	fmt.Println(i18n.T().DeployRollbackSuccess)
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
		fmt.Printf(i18n.T().DeployPruningRelease+"\n", release)
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
	maxRetries := 15 // Default retries
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	healthCheckURL := fmt.Sprintf("http://localhost:%d%s", port, d.config.Service.HealthCheck)

	for i := 0; i < maxRetries; i++ {
		fmt.Print(color.Yellow(fmt.Sprintf(i18n.T().DeployHealthAttempt, i+1, healthCheckURL)))
		resp, err := client.Get(healthCheckURL)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			resp.Body.Close()
			fmt.Println(color.Green(" ✓"))
			return nil // Service is healthy
		}
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Println(color.Red(" ✗"))
		time.Sleep(2 * time.Second) // wait before next retry
	}

	return fmt.Errorf("service not responding at %s after %d attempts", healthCheckURL, maxRetries)
}

func (d *LocalDeployer) setupDirectories() error {
	paths := []string{
		d.config.GetReleasesPath(),
		d.config.GetSharedPath(),
		d.config.GetPidsPath(),
		d.config.GetLogsPath(),
	}
	for _, path := range paths {
		fmt.Printf(i18n.T().DeployEnsuringDir+"\n", path)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (d *LocalDeployer) setupDirectoriesAndRelease(releaseName string, sourceDir string) error {
	if err := d.setupDirectories(); err != nil {
		return err
	}

	fmt.Println(color.Cyan(i18n.T().DeployPopulatingDir))
	releasePath := d.config.GetReleasePathByName(releaseName)
	if sourceDir != "" {
		fmt.Printf(i18n.T().DeployCopyingContent+"\n", sourceDir)
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
		fmt.Printf(i18n.T().DeployLinking+"\n", link, target)
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", item.Name(), err)
		}
	}

	return nil
}

func (d *LocalDeployer) switchSymlink(releaseName string) error {
	releasePath := d.config.GetReleasePathByName(releaseName)
	currentPath := d.config.GetCurrentPath()

	fmt.Printf(i18n.T().DeployPointingSymlink+"\n", releasePath)

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
	return time.Now().Format("20060102-150405")
}

func (d *LocalDeployer) runHooks(hooks []string, hookType string) error {
	if len(hooks) > 0 {
		for _, hook := range hooks {
			if _, err := d.runCommandSyncWithStreaming("sh", "-c", hook); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *LocalDeployer) performHealthCheck(port int) error {
	if d.config.Service.HealthCheck != "" {
		if err := d.waitForService(port); err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
		fmt.Println(color.Green("  - " + i18n.T().DeployHealthPassed))
	}
	return nil
}

func (d *LocalDeployer) runCommandSyncWithStreaming(name string, arg ...string) (string, error) {
	fmt.Println(color.Cyan("  -> Executing: %s %s", name, strings.Join(arg, " ")))
	cmd := exec.Command(name, arg...)
	cmd.Dir = d.config.RootPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for scanner.Scan() {
		fmt.Printf("    %s\n", scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return "", nil
}

func (d *LocalDeployer) runCommandAttachedAsyncWithStreaming(releaseName, command string, env map[string]string) (*exec.Cmd, <-chan error, error) {
	finalCommand := command
	if port, ok := env["PORT"]; ok {
		finalCommand = strings.ReplaceAll(command, "${PORT}", port)
	}

	fullCommand := "sh"
	args := []string{"-c", finalCommand}
	fmt.Println(color.Cyan("  -> Executing: %s %s", fullCommand, strings.Join(args, " ")))
	cmd := exec.Command(fullCommand, args...)
	cmd.Dir = d.config.GetReleasePathByName(releaseName)

	cmd.Env = os.Environ()
	for k, v := range d.config.Deploy.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("could not start command: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Printf("[%s] %s\n", color.Yellow("new-release"), scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("[%s] %s\n", color.Red("new-release-err"), scanner.Text())
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		close(done)
	}()

	return cmd, done, nil
}

func (d *LocalDeployer) stopService() error {
	pidPath := d.resolvePath(d.config.Service.PidFile, "")
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil
	}

	content, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("failed to read pid file: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(content)), ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid pid file format in %s", pidPath)
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid pid in pid file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		return nil
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		log.Printf("Process with PID %d not found, cleaning up stale PID file.", pid)
		os.Remove(pidPath)
		return nil
	}

	log.Printf("Requesting graceful shutdown for process with PID %d...", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to process %d: %w", pid, err)
	}

	timeout := time.Duration(d.config.Service.GracefulTimeout) * time.Second
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		if err := process.Signal(syscall.Signal(0)); err != nil {
			log.Println(color.Green("  -> Service stopped gracefully."))
			os.Remove(pidPath)
			return nil
		}
		log.Printf("  -> Waiting for service to stop... (%v remaining)", time.Until(deadline).Round(time.Second))
	}

	log.Println(color.Yellow("Service did not stop gracefully within the timeout. Forcing shutdown..."))
	if err := process.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to send SIGKILL to process %d: %w", pid, err)
	}
	os.Remove(pidPath)
	log.Println(color.Red("  -> Service forced to stop."))

	return nil
}

func (d *LocalDeployer) startService(releaseName string) error {
	startCmd := d.config.Service.StartCommand
	if startCmd == "" {
		log.Println(color.Yellow("No start_command configured, skipping service start."))
		return nil
	}

	// 1. Define paths
	pidPath := d.resolvePath(d.config.Service.PidFile, releaseName)
	stdoutLogPath := d.resolvePath(d.config.Service.StdoutLog, releaseName)
	stderrLogPath := d.resolvePath(d.config.Service.StderrLog, releaseName)
	releasePath := d.config.GetReleasePathByName(releaseName)
	wrapperPath := filepath.Join(releasePath, "_revlay_starter.sh")

	// 2. Create the wrapper script with robust redirection
	// If a log path is empty, it defaults to /dev/null.
	// If stderr is empty, it redirects to stdout.
	wrapperScript := fmt.Sprintf(`#!/bin/sh
set -e
OUT_LOG_PATH=${1:-/dev/null}
ERR_LOG_PATH=${2:-$OUT_LOG_PATH}
PID_FILE=${3}
shift 3
USER_COMMAND="$@"

echo "Starting service..."
nohup sh -c "$USER_COMMAND" > "$OUT_LOG_PATH" 2> "$ERR_LOG_PATH" &
PID=$!
echo "Service starting with PID: $PID"
echo "$PID:%d" > "$PID_FILE"
`, time.Now().Unix())

	if err := os.WriteFile(wrapperPath, []byte(wrapperScript), 0755); err != nil {
		return fmt.Errorf("failed to create starter script: %w", err)
	}

	// 3. Execute the wrapper script
	cmd := exec.Command(wrapperPath, stdoutLogPath, stderrLogPath, pidPath, startCmd)
	cmd.Dir = releasePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute starter script: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Service start initiated by wrapper script:\n%s", string(output))

	// 4. Wait briefly and check if the process is still running
	// This detects immediate crashes that might occur during startup
	startupDelay := d.config.Service.StartupDelay
	if startupDelay <= 0 {
		startupDelay = 5 // default to 5 seconds if not configured
	}
	log.Printf("Waiting %d seconds to verify service startup...", startupDelay)
	time.Sleep(time.Duration(startupDelay) * time.Second)

	// Check if PID file exists and process is running
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		// Check log files for error messages
		var errorMsg string
		if _, err := os.Stat(stderrLogPath); err == nil {
			// Read the last few lines of the error log
			errorContent, readErr := exec.Command("tail", "-n", "20", stderrLogPath).CombinedOutput()
			if readErr == nil && len(errorContent) > 0 {
				errorMsg = fmt.Sprintf("\nRecent error logs:\n%s", string(errorContent))
			}
		}
		return fmt.Errorf("service failed to start - PID file not found or process died immediately%s", errorMsg)
	}

	// Read PID file and verify process is running
	content, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("failed to read PID file after starting service: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(content)), ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid PID file format after service start")
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid PID in PID file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process with PID %d not found after starting service", pid)
	}

	// Check if process is still running
	if err := process.Signal(syscall.Signal(0)); err != nil {
		// Check log files for error messages
		var errorMsg string
		if _, err := os.Stat(stderrLogPath); err == nil {
			// Read the last few lines of the error log
			errorContent, readErr := exec.Command("tail", "-n", "20", stderrLogPath).CombinedOutput()
			if readErr == nil && len(errorContent) > 0 {
				errorMsg = fmt.Sprintf("\nRecent error logs:\n%s", string(errorContent))
			}
		}
		return fmt.Errorf("service process died shortly after starting%s", errorMsg)
	}

	log.Printf("Service successfully started with PID: %d", pid)
	return nil
}

func (d *LocalDeployer) getCurrentPortFromState() (int, error) {
	stateFile := d.config.GetActivePortPath()
	content, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return d.config.Service.Port, nil // Default to main port if state file doesn't exist
		}
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(content)))
}

func (d *LocalDeployer) writeStateFile(port int) error {
	stateFile := d.config.GetActivePortPath()
	if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
		return fmt.Errorf("could not create state directory: %w", err)
	}
	return os.WriteFile(stateFile, []byte(strconv.Itoa(port)), 0644)
}

// stepLogger helps to print deployment steps with incremental numbers.
type stepLogger struct {
	step int
}

func newStepLogger() *stepLogger {
	return &stepLogger{step: 0}
}

func (l *stepLogger) Print(message string) {
	l.step++
	fmt.Println(color.Cyan(i18n.Sprintf(i18n.T().DeployStep, l.step, message)))
}

func (l *stepLogger) Warn(message string) {
	fmt.Println(color.Yellow(message))
}

func (d *LocalDeployer) preflightChecks(releaseName string) error {
	// Check and create necessary directories
	paths := []string{
		d.config.GetPidsPath(),
		d.config.GetLogsPath(),
	}
	for _, path := range paths {
		// Resolve potential template variables in paths
		resolvedPath, err := d.resolveTemplate(path, releaseName)
		if err != nil {
			return fmt.Errorf("failed to resolve path template %s: %w", path, err)
		}
		// Ensure the directory for the file exists, not the file itself if it's a file path
		dir := filepath.Dir(resolvedPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("  - "+i18n.T().DeployCreatingDir+"\n", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf(i18n.T().DeployDirCreationError, dir, err)
			}
		}
	}

	// TODO: check write permissions for PID and log files

	return nil
}

func (d *LocalDeployer) resolveTemplate(template string, releaseName string) (string, error) {
	template = strings.ReplaceAll(template, "{{.AppName}}", d.config.App.Name)
	template = strings.ReplaceAll(template, "{{.ReleaseName}}", releaseName)
	template = strings.ReplaceAll(template, "{{.Date}}", time.Now().Format("2006-01-02"))
	return template, nil
}

func (d *LocalDeployer) resolvePath(pathTemplate string, releaseName string) string {
	resolved, _ := d.resolveTemplate(pathTemplate, releaseName)
	return filepath.Join(d.config.RootPath, resolved)
}

func copyDirectory(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Lstat(srcPath)
		if err != nil {
			return err
		}

		// Handle by file type
		switch {
		case fileInfo.Mode()&os.ModeSymlink != 0:
			// Handle symlinks
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", srcPath, err)
			}
			if err := os.Symlink(linkTarget, destPath); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", destPath, linkTarget, err)
			}
		case fileInfo.IsDir():
			// Handle directories
			if err := copyDirectory(srcPath, destPath); err != nil {
				return err
			}
		case fileInfo.Mode()&os.ModeNamedPipe != 0, fileInfo.Mode()&os.ModeSocket != 0, fileInfo.Mode()&os.ModeDevice != 0:
			// Skip special files
			log.Printf("Skipping special file: %s (mode: %s)", srcPath, fileInfo.Mode().String())
		default:
			// Regular files
			if err := copyRegularFile(srcPath, destPath, fileInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyRegularFile copies a regular file from src to dest with the given mode
func copyRegularFile(src, dest string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dest, err)
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content from %s to %s: %w", src, dest, err)
	}

	return nil
}
