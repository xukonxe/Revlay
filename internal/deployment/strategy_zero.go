package deployment

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ui"
)

func (d *LocalDeployer) deployZeroDowntime(releaseName string, sourceDir string) error {
	const totalSteps = 7 // 步骤总数，包括清理
	var formatter *ui.DeploymentFormatter
	if d.enableTUI {
		formatter = ui.NewDeploymentFormatter(releaseName, i18n.T().DeployExecZeroDowntime, totalSteps, true)
		formatter.PrintBanner()
	} else {
		fmt.Println(color.Cyan(i18n.T().DeployExecZeroDowntime))
	}

	var log *stepLogger
	if d.enableTUI {
		log = newFormattedStepLogger(formatter)
	} else {
		log = newStepLogger()
	}

	// 统一的错误处理和部署完成逻辑
	handleError := func(err error) error {
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}

	// Step 1: Setup
	log.Print(i18n.T().DeploySetupDirs)
	if err := d.setupDirectoriesAndRelease(releaseName, sourceDir); err != nil {
		return handleError(err)
	}
	if err := d.linkSharedPaths(releaseName); err != nil {
		return handleError(err)
	}
	log.Success(i18n.T().DeploySetupDirsSuccess)

	// Step 2: Determine ports
	log.Print(i18n.T().DeployDeterminePorts)
	oldPort, newPort, err := d.determinePorts()
	if err != nil {
		log.Warn(fmt.Sprintf(i18n.T().DeployDeterminePortsWarn, err))
	}
	log.Print(fmt.Sprintf(i18n.T().DeployCurrentPortInfo, oldPort))
	log.Print(fmt.Sprintf(i18n.T().DeployNewPortInfo, newPort))
	log.Success(i18n.T().DeployDeterminePortsSuccess)

	// Step 3: Start the new version
	log.Print(fmt.Sprintf(i18n.T().DeployStartNewRelease, newPort))
	cmd, processDone, err := d.startNewRelease(releaseName, newPort, formatter)
	if err != nil {
		return handleError(fmt.Errorf(i18n.T().DeployStartNewReleaseFailed, err))
	}
	log.Success(i18n.T().DeployStartNewReleaseSuccess)

	// Step 4: Perform health check
	log.Print(fmt.Sprintf(i18n.T().DeployHealthCheckOnPort, newPort))
	if err := d.monitorHealthCheck(processDone, newPort, cmd); err != nil {
		return handleError(err)
	}
	log.Success(i18n.T().DeployHealthPassed)

	// Step 5: Switch traffic
	log.Print(i18n.T().DeploySwitchProxy)
	if err := d.switchTraffic(releaseName, newPort); err != nil {
		return handleError(err)
	}
	log.Success(fmt.Sprintf(i18n.T().DeploySwitchProxySuccess, newPort))

	// Step 6: Stop old version
	log.Print(fmt.Sprintf(i18n.T().DeployStopOldService, oldPort))
	if err := d.stopOldService(oldPort, releaseName, log); err != nil {
		log.Warn(fmt.Sprintf(i18n.T().DeployStopOldServiceWarn, err))
	} else {
		log.Success(i18n.T().DeployStopOldServiceSuccess)
	}

	// Step 7: Prune old releases
	log.Print(i18n.T().DeployPruning)
	if err := d.Prune(); err != nil {
		log.Warn(fmt.Sprintf(i18n.T().DeployPruningWarn, err))
	} else {
		log.Success(i18n.T().DeployPruningSuccess)
	}

	if formatter != nil {
		formatter.CompleteDeployment(true, "")
	}
	return nil
}

// determinePorts 决定新旧服务的端口
func (d *LocalDeployer) determinePorts() (int, int, error) {
	oldPort, err := d.getCurrentPortFromState()
	if err != nil {
		oldPort = d.config.Service.Port
	}

	newPort := d.config.Service.AltPort
	if oldPort == d.config.Service.AltPort {
		newPort = d.config.Service.Port
	}

	return oldPort, newPort, err
}

// startNewRelease 启动新版本的服务
func (d *LocalDeployer) startNewRelease(releaseName string, newPort int, formatter *ui.DeploymentFormatter) (*exec.Cmd, <-chan error, error) {
	if d.config.Service.StartCommand == "" {
		return nil, nil, fmt.Errorf("start_command not configured")
	}
	env := map[string]string{"PORT": fmt.Sprintf("%d", newPort)}
	return d.runCommandAttachedWithStreaming(releaseName, d.config.Service.StartCommand, env, formatter)
}

// monitorHealthCheck 监控新服务的健康检查
func (d *LocalDeployer) monitorHealthCheck(processDone <-chan error, newPort int, cmd *exec.Cmd) error {
	healthCheckDone := make(chan error, 1)
	go func() {
		healthCheckDone <- d.performHealthCheck(newPort)
	}()

	select {
	case err := <-processDone:
		if err != nil {
			return fmt.Errorf(i18n.T().DeployErrProcExitedEarlyWithError, err)
		}
		return fmt.Errorf(i18n.T().DeployErrProcExitedEarly)
	case err := <-healthCheckDone:
		if err != nil {
			if cmd != nil && cmd.Process != nil {
				cmd.Process.Signal(syscall.SIGTERM)
			}
			return fmt.Errorf(i18n.T().DeployHealthFailed, err)
		}
		return nil
	}
}

// switchTraffic 切换流量到新版本
func (d *LocalDeployer) switchTraffic(releaseName string, newPort int) error {
	if err := d.writeStateFile(newPort); err != nil {
		return fmt.Errorf("failed to write state file to switch traffic: %w", err)
	}
	if err := d.switchSymlink(releaseName); err != nil {
		return fmt.Errorf("failed to switch symlink: %w", err)
	}
	return nil
}

// stopOldService 停止旧版本的服务
func (d *LocalDeployer) stopOldService(oldPort int, currentRelease string, logger *stepLogger) error {
	// 查找旧版本的 PID
	pid, err := d.findPidByPort(oldPort)
	if err != nil {
		return fmt.Errorf(i18n.T().DeployFindOldPidFailed, err)
	}

	if pid == 0 {
		logger.Warn(i18n.T().DeployOldPidNotFound)
		return nil
	}

	// 停止进程
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf(i18n.T().DeployFindOldProcessFailed, pid, err)
	}

	logger.SystemLog(fmt.Sprintf(i18n.T().ServiceGracefulShutdown, pid))
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf(i18n.T().DeployStopOldProcessFailed, pid, err)
	}

	return nil
}

// findPidByPort 根据端口号查找进程ID (这是一个简化的实现)
func (d *LocalDeployer) findPidByPort(port int) (int, error) {
	// 注意: 这是一个简化的、特定于平台的实现，可能不通用
	// 在生产环境中，应该使用更可靠的方法来跟踪进程
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t")
	output, err := cmd.Output()
	if err != nil {
		// 如果 lsof 没有找到任何东西，它会返回一个非零的退出代码
		return 0, nil
	}
	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, nil
	}
	return strconv.Atoi(pidStr)
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
