package deployment

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ui"
)

func (d *LocalDeployer) deployZeroDowntime(releaseName string, sourceDir string) error {
	fmt.Println(color.Cyan(i18n.T().DeployExecZeroDowntime))

	// 创建日志记录器，支持UI模式
	var formatter *ui.DeploymentFormatter
	if d.enableTUI {
		formatter = ui.NewDeploymentFormatter(releaseName, i18n.T().DeployExecZeroDowntime, 6, true)
		formatter.PrintBanner()
	}

	var log *stepLogger
	if d.enableTUI {
		log = newFormattedStepLogger(formatter)
	} else {
		log = newStepLogger()
	}

	// Step 1: Setup
	log.Print(i18n.T().DeploySetupDirs)
	if err := d.setupDirectoriesAndRelease(releaseName, sourceDir); err != nil {
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}
	if err := d.linkSharedPaths(releaseName); err != nil {
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}
	log.Success("目录设置完成")

	// Step 2: Determine ports
	log.Print(i18n.T().DeployDeterminePorts)
	oldPort, err := d.getCurrentPortFromState()
	if err != nil {
		log.Warn(fmt.Sprintf("Could not determine current port: %v. Defaulting to main port.", err))
		oldPort = d.config.Service.Port
	}
	newPort := d.config.Service.AltPort
	if oldPort == d.config.Service.AltPort {
		newPort = d.config.Service.Port
	}
	log.Print(fmt.Sprintf(i18n.T().DeployCurrentPortInfo, oldPort))
	log.Print(fmt.Sprintf(i18n.T().DeployNewPortInfo, newPort))
	log.Success("端口确定完成")

	// Step 3: Start the new version
	log.Print(fmt.Sprintf(i18n.T().DeployStartNewRelease, newPort))
	var newReleaseCmd *exec.Cmd
	var processDone <-chan error
	if d.config.Service.StartCommand != "" {
		env := map[string]string{"PORT": fmt.Sprintf("%d", newPort)}
		var err error
		newReleaseCmd, processDone, err = d.runCommandAttachedAsyncWithStreaming(releaseName, d.config.Service.StartCommand, env)
		if err != nil {
			if formatter != nil {
				formatter.CompleteDeployment(false, err.Error())
			}
			return fmt.Errorf("failed to run start command for new release: %w", err)
		}
	}
	log.Success("新版本已启动")

	// Step 4: Perform health check while monitoring the process
	log.Print(fmt.Sprintf(i18n.T().DeployHealthCheckOnPort, newPort))

	healthCheckDone := make(chan error, 1)
	go func() {
		healthCheckDone <- d.performHealthCheck(newPort)
	}()

	select {
	case err := <-processDone:
		// Process exited before health check completed
		if err != nil {
			log.Error(fmt.Sprintf(i18n.Sprintf(i18n.T().DeployErrProcExitedEarlyWithError, err)))
			if formatter != nil {
				formatter.CompleteDeployment(false, fmt.Sprintf("进程意外退出: %v", err))
			}
			return fmt.Errorf(i18n.Sprintf(i18n.T().DeployErrProcExitedEarlyWithError, err))
		}
		log.Error(i18n.T().DeployErrProcExitedEarly)
		if formatter != nil {
			formatter.CompleteDeployment(false, "进程在健康检查完成前已退出")
		}
		return fmt.Errorf(i18n.T().DeployErrProcExitedEarly)
	case err := <-healthCheckDone:
		if err != nil {
			// Health check failed
			log.Error(fmt.Sprintf(i18n.T().DeployHealthFailed, err))
			// Ensure the newly started process is terminated
			if newReleaseCmd != nil && newReleaseCmd.Process != nil {
				newReleaseCmd.Process.Signal(syscall.SIGTERM)
			}
			if formatter != nil {
				formatter.CompleteDeployment(false, fmt.Sprintf("健康检查失败: %v", err))
			}
			return fmt.Errorf(i18n.Sprintf(i18n.T().DeployHealthFailed, err))
		}
		// Health check passed
		log.Success("健康检查通过")
	}

	// Step 5: Switch traffic
	log.Print(i18n.T().DeploySwitchProxy)
	if err := d.writeStateFile(newPort); err != nil {
		// This is a critical failure. We should try to rollback.
		if formatter != nil {
			formatter.CompleteDeployment(false, fmt.Sprintf("切换流量失败: %v", err))
		}
		return fmt.Errorf("failed to write state file to switch traffic: %w", err)
	}
	if err := d.switchSymlink(releaseName); err != nil {
		if formatter != nil {
			formatter.CompleteDeployment(false, fmt.Sprintf("切换符号链接失败: %v", err))
		}
		return fmt.Errorf("failed to switch symlink: %w", err)
	}
	log.Success(fmt.Sprintf("流量已切换到端口 %d", newPort))

	// Step 6: Stop old version
	log.Print(fmt.Sprintf(i18n.T().DeployStopOldService, oldPort))
	// This requires a more sophisticated `stopService` that can target a specific port/PID.
	// The current `stopService` reads a single PID file. This needs to be adapted for zero-downtime.
	// For now, we assume we can stop the old service.
	// A proper implementation would need to manage PIDs per port.
	log.Warn("Warning: Stopping the old service is not fully implemented for zero-downtime mode yet.")

	// Step 7: Prune old releases
	log.Print(i18n.T().DeployPruning)
	pruneErr := d.Prune()
	if pruneErr != nil {
		log.Warn(fmt.Sprintf("清理旧版本警告: %v", pruneErr))
	} else {
		log.Success("旧版本已清理")
	}

	// 部署成功
	if formatter != nil {
		formatter.CompleteDeployment(true, "")
	}
	return nil
}

// getCurrentPortFromState reads the state file to determine the current active port.
func (d *LocalDeployer) getCurrentPortFromState() (int, error) {
	statePath := d.config.GetActivePortPath()
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return 0, fmt.Errorf("state file not found")
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// writeStateFile writes the current active port to the state file.
func (d *LocalDeployer) writeStateFile(port int) error {
	statePath := d.config.GetActivePortPath()
	return os.WriteFile(statePath, []byte(strconv.Itoa(port)), 0644)
}
