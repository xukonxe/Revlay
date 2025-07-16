package deployment

import (
	"fmt"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ui"
)

func (d *LocalDeployer) deployShortDowntime(releaseName string, sourceDir string) error {
	// 定义总步骤数
	const totalSteps = 7

	// 创建 UI 格式化程序
	var formatter *ui.DeploymentFormatter
	if d.enableTUI {
		formatter = ui.NewDeploymentFormatter(releaseName, i18n.T().DeployExecShortDowntime, totalSteps, true)
		formatter.PrintBanner()
	} else {
		fmt.Println(color.Cyan(i18n.T().DeployExecShortDowntime))
	}

	// 创建日志记录器
	var log *stepLogger
	if d.enableTUI {
		log = newFormattedStepLogger(formatter)
	} else {
		log = newStepLogger()
	}

	// In case of failure, we'll try to rollback to this release
	previousReleaseName, err := d.GetCurrentRelease()
	if err != nil {
		log.Warn(fmt.Sprintf("Could not determine current release for potential rollback: %v", err))
		previousReleaseName = ""
	}

	// Step 1: Run pre-flight checks
	log.Print(i18n.T().DeployPreflightChecks)
	if err := d.preflightChecks(releaseName, log); err != nil {
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}
	log.Success("预检通过")

	// Step 2: Setup directories and copy new release files
	log.Print(i18n.T().DeploySetupDirs)
	if err := d.setupDirectoriesAndRelease(releaseName, sourceDir, log); err != nil {
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}
	if err := d.linkSharedPaths(releaseName, log); err != nil {
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}
	log.Success("目录设置完成")

	// Step 3: Stop the current service
	log.Print(i18n.T().DeployStoppingService)
	if err := d.stopService(log); err != nil {
		log.Warn(fmt.Sprintf(i18n.T().DeployStopServiceFailed, err))
	} else {
		log.Success("服务已停止")
	}

	// Step 4: Activate the new release
	log.Print(i18n.T().DeployActivating)
	if err := d.switchSymlink(releaseName, log); err != nil {
		// If switching the symlink fails, the old service is already stopped.
		// We should try to restart the old service to minimize downtime.
		if previousReleaseName != "" {
			log.Warn("Failed to switch symlink, attempting to restart previous release.")
			d.startService(previousReleaseName, log) // Best effort
		}
		if formatter != nil {
			formatter.CompleteDeployment(false, err.Error())
		}
		return err
	}
	log.Success("新版本已激活")

	// Step 5 & 6: Start new service and perform health check
	startAndCheckError := func() error {
		log.Print(i18n.T().DeployStartingService)
		if err := d.startService(releaseName, log); err != nil {
			return fmt.Errorf(i18n.T().DeployStartServiceFailed, err)
		}
		log.Success("服务已启动")

		log.Print(i18n.T().DeployHealthCheck)
		if err := d.performHealthCheck(d.config.Service.Port); err != nil {
			// Stop the failed service before rolling back
			d.stopService(log)
			return err
		}
		log.Success("健康检查通过")
		return nil
	}()

	if startAndCheckError != nil {
		log.Error(fmt.Sprintf("部署失败: %v", startAndCheckError))

		if previousReleaseName == "" {
			if formatter != nil {
				formatter.CompleteDeployment(false, "部署失败，没有可回滚的版本")
			}
			return fmt.Errorf("deployment failed and no previous release is available to roll back to. The service is stopped")
		}

		log.SystemLog(fmt.Sprintf("尝试回滚到之前的版本: %s", previousReleaseName))

		// Rollback Step 1: Point symlink back to the old release
		if err := d.switchSymlink(previousReleaseName, log); err != nil {
			if formatter != nil {
				formatter.CompleteDeployment(false, "部署失败，回滚也失败")
			}
			return fmt.Errorf("CRITICAL: Deployment failed, and the subsequent rollback also failed when switching symlink. The service may be down. Error: %w", err)
		}

		// Rollback Step 2: Restart the old service
		if err := d.startService(previousReleaseName, log); err != nil {
			if formatter != nil {
				formatter.CompleteDeployment(false, "部署失败，回滚后服务启动失败")
			}
			return fmt.Errorf("CRITICAL: Deployment failed, and the subsequent rollback also failed when restarting the old service. The service may be down. Error: %w", err)
		}

		log.SystemLog(fmt.Sprintf("成功回滚到版本 %s", previousReleaseName))

		if formatter != nil {
			formatter.CompleteDeployment(false, fmt.Sprintf("部署失败，但已成功回滚到 '%s'", previousReleaseName))
		}
		return fmt.Errorf("deployment of '%s' failed, but successfully rolled back to '%s'", releaseName, previousReleaseName)
	}

	// Step 7: Prune old releases (不显示为主要步骤，作为附加操作)
	log.Print(i18n.T().DeployPruning)
	pruneErr := d.Prune(log)
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
