package deployment

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/config"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ui"
)

// ServiceAlreadyRunningError is returned when a service is already running.
type ServiceAlreadyRunningError struct {
	PID int
}

func (e *ServiceAlreadyRunningError) Error() string {
	return fmt.Sprintf("service already running with PID %d", e.PID)
}

// Deployer defines the interface for deployment operations.
type Deployer interface {
	Deploy(releaseName string, sourceDir string) error
	Rollback(releaseName string) error
	ListReleases() ([]string, error)
	GetCurrentRelease() (string, error)
	Prune(logger *stepLogger) error
	StartService(releaseName string) error
	StopService() error
}

// Release represents a deployment release.
type Release struct {
	Name    string
	Path    string
	Current bool
}

// LocalDeployer handles deployments on the local machine.
type LocalDeployer struct {
	config    *config.Config
	enableTUI bool
}

// NewLocalDeployer creates a new deployer for local operations.
func NewLocalDeployer(cfg *config.Config) Deployer {
	return &LocalDeployer{
		config: cfg,
	}
}

// NewLocalDeployerWithOptions 创建一个新的本地部署器，并支持额外选项
func NewLocalDeployerWithOptions(cfg *config.Config, enableTUI bool) Deployer {
	return &LocalDeployer{
		config:    cfg,
		enableTUI: enableTUI,
	}
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

	// Run pre-deployment hooks
	if err := d.runHooks(d.config.Hooks.PreDeploy, "pre-deploy"); err != nil {
		return fmt.Errorf("pre-deploy hook failed: %w", err)
	}

	var deployErr error
	switch d.config.Deploy.Mode {
	case config.ZeroDowntimeMode:
		deployErr = d.deployZeroDowntime(releaseName, sourceDir)
	case config.ShortDowntimeMode:
		deployErr = d.deployShortDowntime(releaseName, sourceDir)
	default:
		log.Printf("Unknown deployment mode '%s', falling back to short_downtime.", d.config.Deploy.Mode)
		deployErr = d.deployShortDowntime(releaseName, sourceDir)
	}

	if deployErr != nil {
		// Run post-deployment hooks even if deploy failed (for cleanup)
		if err := d.runHooks(d.config.Hooks.PostDeploy, "post-deploy"); err != nil {
			log.Printf("post-deploy hook failed after a failed deployment: %v", err)
		}
		return deployErr
	}

	// Run post-deployment hooks
	if err := d.runHooks(d.config.Hooks.PostDeploy, "post-deploy"); err != nil {
		return fmt.Errorf("post-deploy hook failed: %w", err)
	}

	return nil
}

func (d *LocalDeployer) Rollback(releaseName string) error {
	fmt.Println(color.Cyan(i18n.T().RollbackStarting, releaseName))

	// 1. Get list of releases
	releases, err := d.ListReleases()
	if err != nil {
		return err
	}
	var found bool
	for _, r := range releases {
		if r == releaseName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf(i18n.T().ErrorReleaseNotFound, releaseName)
	}

	// 2. Stop current service
	fmt.Println("  -> Stopping current service...")
	if err := d.stopService(nil); err != nil {
		fmt.Printf(color.Yellow(i18n.T().DeployStopServiceFailed, err))
	}

	// 3. Switch symlink
	fmt.Println("  -> Activating rollback release...")
	if err := d.switchSymlink(releaseName, nil); err != nil {
		return err
	}

	// 4. Start service
	fmt.Println("  -> Starting service...")
	if err := d.startService(releaseName, nil); err != nil {
		return err
	}

	fmt.Println(color.Green(i18n.T().RollbackSuccess, releaseName))
	return nil
}

func GenerateReleaseTimestamp() string {
	return time.Now().UTC().Format("20060102150405")
}

func (d *LocalDeployer) runHooks(hooks []string, hookType string) error {
	if len(hooks) == 0 {
		return nil
	}
	fmt.Println(color.Cyan("  -> Running %s hooks...", hookType))
	currentPath := d.config.GetCurrentPath()

	for _, hook := range hooks {
		resolvedHook, err := d.resolveTemplate(hook, "") // releaseName is not relevant for all hooks
		if err != nil {
			return fmt.Errorf("could not resolve hook template '%s': %w", hook, err)
		}
		parts := strings.Fields(resolvedHook)
		if len(parts) == 0 {
			continue
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = currentPath // Run hook in the context of the current code
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("hook '%s' failed: %s", resolvedHook, output)
		}
	}
	return nil
}

func (d *LocalDeployer) runCommandAttachedAsync(releaseName, command string, env map[string]string) (*exec.Cmd, <-chan error, error) {
	cmdStr, err := d.resolveTemplate(command, releaseName)
	if err != nil {
		return nil, nil, fmt.Errorf("could not resolve command template: %w", err)
	}

	cmdParts := strings.Fields(cmdStr)
	if len(cmdParts) == 0 {
		return nil, nil, fmt.Errorf("empty command after resolving template")
	}

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = d.config.GetReleasePathByName(releaseName)

	// Set up environment variables
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Goroutine to stream stdout and stderr
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("[%s-out] %s", releaseName, scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[%s-err] %s", releaseName, scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start command '%s': %w", cmdStr, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	return cmd, done, nil
}

func (d *LocalDeployer) runCommandAttachedWithStreaming(releaseName, command string, env map[string]string, formatter *ui.DeploymentFormatter) (*exec.Cmd, <-chan error, error) {
	cmdStr, err := d.resolveTemplate(command, releaseName)
	if err != nil {
		return nil, nil, fmt.Errorf("could not resolve command template: %w", err)
	}

	cmdParts := strings.Fields(cmdStr)
	if len(cmdParts) == 0 {
		return nil, nil, fmt.Errorf("empty command after resolving template")
	}

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = d.config.GetReleasePathByName(releaseName)

	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if formatter != nil {
		formatter.StartStreaming(releaseName)
		go streamOutput(stdout, releaseName, "out", formatter)
		go streamOutput(stderr, releaseName, "err", formatter)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start command '%s': %w", cmdStr, err)
	}

	processDone := make(chan error, 1)
	go func() {
		processDone <- cmd.Wait()
		if formatter != nil {
			formatter.StopStreaming()
		}
	}()

	return cmd, processDone, nil
}

func streamOutput(reader io.Reader, releaseName, streamType string, formatter *ui.DeploymentFormatter) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if formatter != nil {
			formatter.StreamLog(releaseName, streamType, scanner.Text())
		}
	}
}

func (d *LocalDeployer) resolveTemplate(template string, releaseName string) (string, error) {
	// A simple key-value replacer. More complex templating can be added later.
	if releaseName == "" {
		var err error
		releaseName, err = d.GetCurrentRelease()
		if err != nil {
			// It might be the first deployment, so no current release exists.
			// In this case, we can't resolve {{release}} but other templates might be fine.
			releaseName = ""
		}
	}
	template = strings.ReplaceAll(template, "{{.AppName}}", d.config.App.Name)
	template = strings.ReplaceAll(template, "{{.ReleaseName}}", releaseName)
	template = strings.ReplaceAll(template, "{{release}}", releaseName) // Keep for compatibility
	template = strings.ReplaceAll(template, "{{current_path}}", d.config.GetCurrentPath())
	template = strings.ReplaceAll(template, "{{.Date}}", time.Now().Format("2006-01-02"))

	return template, nil
}
