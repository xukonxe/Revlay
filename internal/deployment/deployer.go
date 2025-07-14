package deployment

import (
	"bufio"
	"fmt"
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

	// Step 1: Setup directories
	fmt.Println(color.Cyan(i18n.T().DeployStep, 1, i18n.T().DeploySetupDirs))
	if err := d.setupDirectoriesAndRelease(releaseName, sourceDir); err != nil {
		return err
	}
	if err := d.linkSharedPaths(releaseName); err != nil {
		return err
	}

	// Step 2: Stop the current service
	fmt.Println(color.Cyan(i18n.T().DeployStep, 2, "Stopping current service..."))
	if d.config.Service.StopCommand != "" {
		env := map[string]string{"PORT": fmt.Sprintf("%d", d.config.Service.Port)}
		if _, err := d.runCommandSync(releaseName, d.config.Service.StopCommand, env); err != nil {
			log.Print(color.Yellow(fmt.Sprintf("Warning: failed to run stop command: %v.", err)))
		}
	}

	// Step 3: Activate new release
	fmt.Println(color.Cyan(i18n.T().DeployStep, 3, i18n.T().DeployActivating))
	if err := d.switchSymlink(releaseName); err != nil {
		return err
	}

	// Step 4: Start the new service
	fmt.Println(color.Cyan(i18n.T().DeployStep, 4, "Starting new service..."))
	if d.config.Service.StartCommand != "" {
		env := map[string]string{"PORT": fmt.Sprintf("%d", d.config.Service.Port)}
		if _, err := d.runCommandAsync(releaseName, d.config.Service.StartCommand, env); err != nil {
			return fmt.Errorf("failed to run start command: %w", err)
		}
	}

	// Step 5: Perform health check
	fmt.Println(color.Cyan(i18n.T().DeployStep, 5, i18n.T().DeployHealthCheck))
	if err := d.performHealthCheck(d.config.Service.Port); err != nil {
		return err
	}

	// Step 6: Prune old releases
	fmt.Println(color.Cyan(i18n.T().DeployStep, 6, i18n.T().DeployPruning))
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
	d.stopService(releaseName, oldPort)

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
	fmt.Println(color.Cyan(i18n.T().DeploySetupDirs))
	if err := d.setupDirectories(); err != nil {
		return err
	}

	fmt.Println(color.Cyan(i18n.T().DeployPopulatingDir))
	releasePath := d.config.GetReleasePathByName(releaseName)
	if sourceDir != "" {
		fmt.Printf(i18n.T().DeployMovingContent+"\n", sourceDir)
		if err := os.Rename(sourceDir, releasePath); err != nil {
			fmt.Println(color.Yellow(i18n.T().DeployRenameFailed))
			if _, err := d.runLocalCommand("cp", "-r", sourceDir, releasePath); err != nil {
				return fmt.Errorf("failed to copy from source directory %s: %w", sourceDir, err)
			}
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
		// No need to print this sub-step, the main steps are enough
		// fmt.Print(color.Cyan(fmt.Sprintf("Running %s hooks...\n", hookType)))
		for _, hook := range hooks {
			if _, err := d.runLocalCommand("sh", "-c", hook); err != nil {
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

func (d *LocalDeployer) runCommandSync(releaseName, command string, env map[string]string) (string, error) {
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(i18n.T().DeployCmdExecFailed, err, output)
	}
	return string(output), nil
}

func (d *LocalDeployer) runCommandAsync(releaseName, command string, env map[string]string) (*exec.Cmd, error) {
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	// Detach the process from the deployer
	if err := cmd.Process.Release(); err != nil {
		log.Printf("Warning: failed to release process: %v", err)
	}
	return cmd, nil
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

func (d *LocalDeployer) stopService(releaseName string, port int) {
	if d.config.Service.StopCommand != "" {
		// No need to print this sub-step, the main steps are enough
		// fmt.Print(color.Cyan(fmt.Sprintf("Stopping service on port %d...\n", port)))
		env := map[string]string{"PORT": fmt.Sprintf("%d", port)}
		if _, err := d.runCommandSync(releaseName, d.config.Service.StopCommand, env); err != nil {
			log.Print(color.Yellow(fmt.Sprintf("Warning: failed to stop service on port %d: %v", port, err)))
		}
	}
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
