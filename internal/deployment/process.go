package deployment

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
)

// waitForService waits for a service to become available on a given port.
func (d *LocalDeployer) waitForService(port int) error {
	maxRetries := d.config.Service.HealthCheckRetries
	if maxRetries <= 0 {
		maxRetries = 10 // Default retries
	}

	timeout := d.config.Service.HealthCheckTimeout
	if timeout <= 0 {
		timeout = 5 // Default timeout in seconds
	}

	interval := d.config.Service.HealthCheckInterval
	if interval <= 0 {
		interval = 2 // Default interval in seconds
	}

	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	healthCheckURL := fmt.Sprintf("http://localhost:%d%s", port, d.config.Service.HealthCheck)

	for i := 0; i < maxRetries; i++ {
		log.Print(i18n.Sprintf(i18n.T().DeployHealthAttempt, i+1, healthCheckURL))
		resp, err := client.Get(healthCheckURL)
		if err == nil {
			// Ensure body is closed to prevent resource leaks
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				log.Println(color.Green(i18n.T().DeployHealthPassed))
				return nil // Service is healthy
			}
		}

		if i < maxRetries-1 {
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}

	return fmt.Errorf("service at %s did not respond after %d attempts", healthCheckURL, maxRetries)
}

// performHealthCheck performs a health check on the given port.
func (d *LocalDeployer) performHealthCheck(port int) error {
	return d.waitForService(port)
}

// stopService stops the running service by reading the PID file.
// This is an internal function that doesn't expose itself via the Deployer interface.
// The public one is StopService.
func (d *LocalDeployer) stopService(logger *stepLogger) error {
	pidPath := d.resolvePath(d.config.Service.PidFile, "")
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		// 使用 logger 而不是 log.Println
		if logger != nil {
			logger.SystemLog("No PID file found, service may not be running.")
		} else {
			log.Println("No PID file found, service may not be running.")
		}
		return nil
	}

	content, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("could not read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(content))
	var pid int

	// 兼容旧格式 (PID:timestamp)
	if strings.Contains(pidStr, ":") {
		pidParts := strings.Split(pidStr, ":")
		if len(pidParts) > 0 {
			pid, _ = strconv.Atoi(pidParts[0])
		}
	} else {
		pid, _ = strconv.Atoi(pidStr)
	}

	if pid <= 0 {
		return fmt.Errorf("invalid PID in file: %s", pidPath)
	}

	// 使用 logger 而不是 log.Println
	if logger != nil {
		logger.SystemLog(fmt.Sprintf(i18n.T().ServiceGracefulShutdown, pid))
	} else {
		log.Println(color.Yellow(i18n.T().ServiceGracefulShutdown, pid))
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process with PID %d: %w", pid, err)
	}

	// Send the SIGTERM signal
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("could not send SIGTERM to process: %w", err)
	}

	// 给予进程一些时间来清理退出
	gracePeriod := 10 * time.Second
	done := make(chan error)

	go func() {
		state, err := process.Wait()
		if err != nil {
			done <- err
			return
		}
		done <- nil
		// 使用 logger 而不是 log.Println
		if logger != nil {
			if state.Success() {
				logger.SystemLog("Service stopped gracefully.")
			} else {
				logger.SystemLog(fmt.Sprintf("Service exited with status: %v", state.String()))
			}
		} else {
			if state.Success() {
				log.Println("Service stopped gracefully.")
			} else {
				log.Printf("Service exited with status: %v", state.String())
			}
		}
	}()

	select {
	case err := <-done:
		os.Remove(pidPath) // 进程结束后，清理PID文件
		return err
	case <-time.After(gracePeriod):
		// 超时，强制终止进程
		// 使用 logger 而不是 log.Println
		if logger != nil {
			logger.SystemLog("Service did not stop gracefully. Forcing shutdown...")
		} else {
			log.Println(color.Red("Service did not stop gracefully. Forcing shutdown..."))
		}
		err := process.Kill()
		os.Remove(pidPath) // 进程结束后，清理PID文件
		return err
	}
}

// StopService is the public method to stop the service.
func (d *LocalDeployer) StopService() error {
	return d.stopService(nil) // Pass nil for now, as stepLogger is not directly available here
}

// startService starts the service for a given release.
func (d *LocalDeployer) startService(releaseName string, logger *stepLogger) error {
	// Check if service is already running by checking the PID file
	pidPath := d.resolvePath(d.config.Service.PidFile, releaseName)
	if _, err := os.Stat(pidPath); err == nil {
		content, err := os.ReadFile(pidPath)
		if err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(content)))
			// Fallback for old format PID:timestamp
			if parts := strings.Split(strings.TrimSpace(string(content)), ":"); len(parts) == 2 {
				pid, _ = strconv.Atoi(parts[0])
			}
			if process, err := os.FindProcess(pid); err == nil {
				if err := process.Signal(syscall.Signal(0)); err == nil {
					return &ServiceAlreadyRunningError{PID: pid}
				}
			}
		}
		log.Println(color.Yellow(i18n.T().ServiceStalePidFile))
		os.Remove(pidPath)
	}

	startCmd := d.config.Service.StartCommand
	if startCmd == "" {
		log.Println(color.Yellow("No start_command configured, skipping service start."))
		return nil
	}

	// Paths
	stdoutLogPath := d.resolvePath(d.config.Service.StdoutLog, releaseName)
	stderrLogPath := d.resolvePath(d.config.Service.StderrLog, releaseName)
	releasePath := d.config.GetReleasePathByName(releaseName)

	cmd := exec.Command("sh", "-c", startCmd)
	cmd.Dir = releasePath
	cmd.Env = os.Environ()
	for key, value := range d.config.Deploy.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", d.config.Service.Port))

	// Ensure log directories exist
	for _, logPath := range []string{stdoutLogPath, stderrLogPath} {
		if logPath != "" {
			if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
				return fmt.Errorf("failed to create log directory for %s: %w", logPath, err)
			}
		}
	}

	// Redirect stdout/stderr
	cmd.Stdout, _ = os.OpenFile(stdoutLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if stderrLogPath != "" && stderrLogPath != stdoutLogPath {
		cmd.Stderr, _ = os.OpenFile(stderrLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	} else {
		cmd.Stderr = cmd.Stdout
	}

	// Start the command in a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Write the PID to a file
	pid := cmd.Process.Pid
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write pid file: %w", err)
	}

	// 在最后修改日志输出
	if logger != nil {
		logger.SystemLog(fmt.Sprintf(i18n.T().ServiceStartInitiated, cmd.Process.Pid, stdoutLogPath))
	} else {
		log.Println(color.Green(i18n.T().ServiceStartInitiated, cmd.Process.Pid, stdoutLogPath))
	}

	startupDelay := d.config.Service.StartupDelay
	if startupDelay > 0 {
		time.Sleep(time.Duration(startupDelay) * time.Second)
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("service process died shortly after starting")
		}
	}

	return nil
}

// StartService is the public method to start the service.
func (d *LocalDeployer) StartService(releaseName string) error {
	return d.startService(releaseName, nil) // Pass nil for now, as stepLogger is not directly available here
}
