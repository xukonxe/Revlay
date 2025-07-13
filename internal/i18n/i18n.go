package i18n

import (
	"fmt"
	"os"
)

// Language represents supported languages
type Language string

const (
	Chinese Language = "zh"
	English Language = "en"
)

// Messages holds all translatable strings
type Messages struct {
	// CLI Messages
	AppShortDesc   string
	AppLongDesc    string
	AppVersion     string
	ConfigFileFlag string
	LanguageFlag   string

	// Init Command
	InitShortDesc     string
	InitLongDesc      string
	InitSuccess       string
	InitFailed        string
	InitNameFlag      string
	InitPathFlag      string
	InitDirectoryFlag string // I'm adding this line
	InitPromptName    string
	InitPromptPath    string

	// Deploy Command
	DeployShortDesc   string
	DeployLongDesc    string
	DeployStarting    string
	DeployDryRunMode  string
	DeploySSHTest     string
	DeploySSHSuccess  string
	DeployInProgress  string
	DeploySuccess     string
	DeployFailed      string
	DeployDryRunFlag  string
	DeployReleaseLive string
	DeployDryRunPlan  string

	// Rollback Command
	RollbackShortDesc  string
	RollbackLongDesc   string
	RollbackSuccess    string
	RollbackFailed     string
	RollbackToRelease  string
	RollbackNoReleases string
	RollbackStarting   string

	// Releases Command
	ReleasesShortDesc  string
	ReleasesLongDesc   string
	ReleasesListHeader string
	ReleasesNoReleases string
	ReleasesCurrent    string
	ReleasesHeader     string
	ErrorReleasesList  string

	// Status Command
	StatusShortDesc      string
	StatusLongDesc       string
	StatusCurrentRelease string
	StatusNoRelease      string
	StatusAppName        string
	StatusDeployPath     string
	StatusServerInfo     string

	// Push Command
	PushShortDesc        string
	PushLongDesc         string
	PushStarting         string
	PushCheckingRemote   string
	PushRemoteFound      string
	PushCreatingTempDir  string
	PushTempDirCreated   string
	PushCleaningUp       string
	PushCleanupFailed    string
	PushCleanupComplete  string
	PushSyncingFiles     string
	PushSyncComplete     string
	PushTriggeringDeploy string
	PushComplete         string

	// Dry Run Messages
	DryRunPlan           string
	DryRunApplication    string
	DryRunServer         string
	DryRunRelease        string
	DryRunDeployPath     string
	DryRunReleasesPath   string
	DryRunSharedPath     string
	DryRunCurrentPath    string
	DryRunReleasePathFmt string
	DryRunDirStructure   string
	DryRunHooks          string
	DryRunPreDeploy      string
	DryRunPostDeploy     string
	DryRunKeepReleases   string

	// Error Messages
	ErrorConfigNotFound  string
	ErrorConfigLoad      string
	ErrorSSHConnect      string
	ErrorSSHTest         string
	ErrorDeployment      string
	ErrorRollback        string
	ErrorNoReleases      string
	ErrorReleaseNotFound string

	// Deployment Modes
	DeploymentMode     string
	ZeroDowntime       string
	ShortDowntime      string
	DeploymentModeDesc string

	// Service Management
	ServiceManagement   string
	ServicePort         string
	ServiceCommand      string
	ServiceHealthCheck  string
	ServiceRestartDelay string
}

var currentLanguage Language = Chinese
var messages Messages

// Chinese messages
var chineseMessages = Messages{
	AppShortDesc:   "现代化、快速、无依赖的部署工具",
	AppLongDesc:    `Revlay是一个现代化的部署工具，提供原子部署、零停机部署和传统服务器部署的轻松回滚功能。\n\n它使用结构化的目录布局，包含releases、shared文件和原子符号链接切换，确保可靠的部署。`,
	AppVersion:     "版本",
	ConfigFileFlag: "配置文件 (默认为 revlay.yml)",
	LanguageFlag:   "语言设置 (zh|en)",

	InitShortDesc:     "使用 revlay.yml 文件初始化一个新项目",
	InitLongDesc:      `init 命令在当前或指定目录中创建一个新的 revlay.yml 配置文件。`,
	InitNameFlag:      "应用名称",
	InitPathFlag:      "服务器上的部署路径",
	InitDirectoryFlag: "用于初始化的目标目录",
	InitPromptName:    "应用名称",
	InitPromptPath:    "部署路径",
	InitFailed:        "初始化失败: %v",
	InitSuccess:       "配置文件已创建于 %s",

	// deploy command
	DeployShortDesc:   "部署新版本",
	DeployLongDesc:    "部署新版本到服务器。\n\n如果没有提供版本名称，将自动生成基于时间戳的名称。\n此命令将创建新的版本目录，链接共享路径，并切换current符号链接到新版本。",
	DeployStarting:    "🚀 开始部署版本：%s",
	DeployDryRunMode:  "🔍 预览模式 - 不会进行实际更改",
	DeploySSHTest:     "🔗 测试SSH连接...",
	DeploySSHSuccess:  "✓ SSH连接成功",
	DeployInProgress:  "📦 正在部署版本...",
	DeploySuccess:     "✓ 部署成功完成",
	DeployFailed:      "部署失败：%v",
	DeployDryRunFlag:  "显示将要执行的操作，但不实际部署",
	DeployReleaseLive: "✓ 版本 %s 已在 %s 上线",
	DeployDryRunPlan:  "部署计划:",

	// releases command
	ReleasesShortDesc:  "列出所有已部署的版本",
	ReleasesLongDesc:   "列出在 releases 目录中找到的所有版本。",
	ReleasesListHeader: "📋 已部署的版本：",
	ReleasesNoReleases: "未找到任何版本。",
	ReleasesCurrent:    " (当前)",
	ReleasesHeader:     "%-18s %s",
	ErrorReleasesList:  "列出版本失败: %v",

	// rollback command
	RollbackShortDesc:  "回滚到上一个版本",
	RollbackLongDesc:   "通过切换 'current' 符号链接将应用程序回滚到指定的版本。",
	RollbackStarting:   "正在回滚到版本 %s...",
	RollbackSuccess:    "成功回滚到 %s。",
	RollbackFailed:     "回滚失败：%v",
	RollbackToRelease:  "🔄 回滚到版本：%s",
	RollbackNoReleases: "没有找到可回滚的版本",

	// Status Command
	StatusShortDesc:      "显示部署状态",
	StatusLongDesc:       "显示当前部署的版本和其他状态信息。",
	StatusCurrentRelease: "当前版本：%s",
	StatusNoRelease:      "没有激活的版本",
	StatusAppName:        "应用名称：%s",
	StatusDeployPath:     "部署路径：%s",
	StatusServerInfo:     "服务器：%s@%s:%d",

	// Push Command
	PushShortDesc:        "推送本地目录到远程服务器并部署",
	PushLongDesc:         `此命令使用rsync将本地目录推送到远程服务器，并在远程机器上触发'revlay deploy'。\n\n它通过打包、传输和在单个步骤中激活新版本来简化部署过程。`,
	PushStarting:         "🚀 开始推送到 %s 应用 '%s'...",
	PushCheckingRemote:   "🔎 检查远程环境...",
	PushRemoteFound:      "✅ 远程'revlay'命令已找到。",
	PushCreatingTempDir:  "📁 在远程创建临时目录...",
	PushTempDirCreated:   "✅ 已创建临时目录: %s",
	PushCleaningUp:       "\n🧹 清理远程临时目录...",
	PushCleanupFailed:    "⚠️ 清理临时目录 %s 失败: %v",
	PushCleanupComplete:  "✅ 清理完成。",
	PushSyncingFiles:     "🚚 同步文件到 %s...",
	PushSyncComplete:     "✅ 文件同步成功完成。",
	PushTriggeringDeploy: "🚢 正在为应用 '%s' 触发远程部署...",
	PushComplete:         "\n🎉 推送和部署成功完成！",

	DryRunPlan:           "📋 部署计划：",
	DryRunApplication:    "应用",
	DryRunServer:         "服务器",
	DryRunRelease:        "版本",
	DryRunDeployPath:     "部署路径",
	DryRunReleasesPath:   "版本路径",
	DryRunSharedPath:     "共享路径",
	DryRunCurrentPath:    "当前路径",
	DryRunReleasePathFmt: "版本路径",
	DryRunDirStructure:   "📂 将要创建的目录结构：",
	DryRunHooks:          "🪝 将要执行的钩子：",
	DryRunPreDeploy:      "部署前",
	DryRunPostDeploy:     "部署后",
	DryRunKeepReleases:   "🧹 保留 %d 个版本（旧版本将被清理）",

	ErrorConfigNotFound:  "未找到配置文件: %s",
	ErrorConfigLoad:      "加载配置失败：%v",
	ErrorSSHConnect:      "连接服务器失败：%v",
	ErrorSSHTest:         "SSH连接测试失败：%v",
	ErrorDeployment:      "部署失败：%v",
	ErrorRollback:        "回滚失败：%v",
	ErrorNoReleases:      "没有找到可用的版本",
	ErrorReleaseNotFound: "版本 %s 不存在",

	DeploymentMode:     "部署模式",
	ZeroDowntime:       "零停机部署",
	ShortDowntime:      "短停机部署",
	DeploymentModeDesc: "部署模式说明",

	ServiceManagement:   "服务管理",
	ServicePort:         "服务端口",
	ServiceCommand:      "服务命令",
	ServiceHealthCheck:  "健康检查",
	ServiceRestartDelay: "重启延迟",
}

// English messages
var englishMessages = Messages{
	AppShortDesc:   "A modern, fast, dependency-free deployment and server lifecycle management tool.",
	AppLongDesc:    `Revlay is a command-line tool for deploying and managing web applications.`,
	ConfigFileFlag: "Path to config file (default is revlay.yml)",
	LanguageFlag:   "Language for output (e.g., 'en', 'zh')",

	// init command
	InitShortDesc:     "Initialize a new project with a revlay.yml file",
	InitLongDesc:      `The init command creates a new revlay.yml configuration file in the current or specified directory.`,
	InitNameFlag:      "Application name",
	InitPathFlag:      "Deployment path on the server",
	InitDirectoryFlag: "Target directory for initialization",
	InitPromptName:    "Application name",
	InitPromptPath:    "Deployment path",
	InitFailed:        "Initialization failed: %v",
	InitSuccess:       "Configuration file created at %s",

	// deploy command
	DeployShortDesc:   "Deploy the application to the server",
	DeployLongDesc:    "Deploy a new release to the server.\n\nIf no release name is provided, a timestamp-based name will be generated.\nThis command will create a new release directory, link shared paths,\nand switch the current symlink to the new release.",
	DeployStarting:    "🚀 Starting deployment of release: %s",
	DeployDryRunMode:  "🔍 DRY RUN MODE - No actual changes will be made",
	DeploySSHTest:     "🔗 Testing SSH connection...",
	DeploySSHSuccess:  "✓ SSH connection successful",
	DeployInProgress:  "📦 Deploying release...",
	DeploySuccess:     "✓ Deployment completed successfully",
	DeployFailed:      "Deployment failed: %v",
	DeployDryRunFlag:  "Show what would be done without actually deploying",
	DeployReleaseLive: "✓ Release %s is now live at %s",
	DeployDryRunPlan:  "Deployment Plan:",

	// releases command
	ReleasesShortDesc:  "List all deployed releases",
	ReleasesLongDesc:   "Lists all releases found in the releases directory.",
	ReleasesListHeader: "📋 Deployed releases:",
	ReleasesNoReleases: "No releases found.",
	ReleasesCurrent:    " (current)",
	ReleasesHeader:     "%-18s %s",
	ErrorReleasesList:  "Failed to list releases: %v",

	// rollback command
	RollbackShortDesc:  "Rollback to a previous release",
	RollbackLongDesc:   "Rolls back the application to a specified release by switching the 'current' symlink.",
	RollbackStarting:   "Rolling back to release %s...",
	RollbackSuccess:    "Successfully rolled back to %s.",
	RollbackFailed:     "Rollback failed: %v",
	RollbackToRelease:  "🔄 Rolling back to release: %s",
	RollbackNoReleases: "No releases found to rollback to",

	// Status Command
	StatusShortDesc:      "Show the status of the deployment",
	StatusLongDesc:       "Displays the current deployed release and other status information.",
	StatusCurrentRelease: "Current release: %s",
	StatusNoRelease:      "No active release",
	StatusAppName:        "Application: %s",
	StatusDeployPath:     "Deploy path: %s",
	StatusServerInfo:     "Server: %s@%s:%d",

	// Push Command
	PushShortDesc:        "Push local directory to remote and deploy",
	PushLongDesc:         `This command uses rsync to push a local directory to a remote server and then triggers 'revlay deploy' on the remote machine.\n\nIt streamlines the deployment process by packaging, transferring, and activating a new release in a single step.`,
	PushStarting:         "🚀 Starting push to %s for app '%s'...",
	PushCheckingRemote:   "🔎 Checking remote environment...",
	PushRemoteFound:      "✅ Remote 'revlay' command found.",
	PushCreatingTempDir:  "📁 Creating temporary directory on remote...",
	PushTempDirCreated:   "✅ Created temporary directory: %s",
	PushCleaningUp:       "\n🧹 Cleaning up temporary directory on remote...",
	PushCleanupFailed:    "⚠️ Failed to clean up temporary directory %s: %v",
	PushCleanupComplete:  "✅ Cleanup complete.",
	PushSyncingFiles:     "🚚 Syncing files to %s...",
	PushSyncComplete:     "✅ File sync completed successfully.",
	PushTriggeringDeploy: "🚢 Triggering remote deployment for app '%s'...",
	PushComplete:         "\n🎉 Push and deploy completed successfully!",

	DryRunPlan:           "📋 Deployment plan:",
	DryRunApplication:    "Application",
	DryRunServer:         "Server",
	DryRunRelease:        "Release",
	DryRunDeployPath:     "Deploy path",
	DryRunReleasesPath:   "Releases path",
	DryRunSharedPath:     "Shared path",
	DryRunCurrentPath:    "Current path",
	DryRunReleasePathFmt: "Release path",
	DryRunDirStructure:   "Directory structure to be created:",
	DryRunHooks:          "Hooks",
	DryRunPreDeploy:      "Pre-deploy",
	DryRunPostDeploy:     "Post-deploy",
	DryRunKeepReleases:   "🧹 Keep %d releases (older ones will be cleaned up)",

	ErrorConfigNotFound:  "config file %s not found, run 'revlay init' first",
	ErrorConfigLoad:      "failed to load config: %v",
	ErrorSSHConnect:      "failed to connect to server: %v",
	ErrorSSHTest:         "SSH connection test failed: %v",
	ErrorDeployment:      "deployment failed: %v",
	ErrorRollback:        "rollback failed: %v",
	ErrorNoReleases:      "no releases found",
	ErrorReleaseNotFound: "release %s does not exist",

	DeploymentMode:     "Deployment Mode",
	ZeroDowntime:       "Zero Downtime Deployment",
	ShortDowntime:      "Short Downtime Deployment",
	DeploymentModeDesc: "Deployment Mode Description",

	ServiceManagement:   "Service Management",
	ServicePort:         "Service Port",
	ServiceCommand:      "Service Command",
	ServiceHealthCheck:  "Health Check",
	ServiceRestartDelay: "Restart Delay",
}

// SetLanguage sets the current language
func SetLanguage(lang Language) {
	currentLanguage = lang
	switch lang {
	case English:
		messages = englishMessages
	case Chinese:
		messages = chineseMessages
	default:
		messages = chineseMessages
	}
}

// GetLanguage returns the current language
func GetLanguage() Language {
	return currentLanguage
}

// GetMessages returns the current messages
func GetMessages() Messages {
	return messages
}

// T returns a translated message (convenience function)
func T() Messages {
	return messages
}

// InitLanguage initializes the language based on environment or flag
func InitLanguage(langFlag string) {
	var lang Language

	if langFlag != "" {
		switch langFlag {
		case "en", "english":
			lang = English
		case "zh", "chinese":
			lang = Chinese
		default:
			lang = Chinese
		}
	} else {
		// Check environment variable
		if envLang := os.Getenv("REVLAY_LANG"); envLang != "" {
			switch envLang {
			case "en", "english":
				lang = English
			case "zh", "chinese":
				lang = Chinese
			default:
				lang = Chinese
			}
		} else {
			// Default to Chinese
			lang = Chinese
		}
	}

	SetLanguage(lang)
}

// Sprintf formats a string with the current language
func Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// Initialize default language on package load
func init() {
	SetLanguage(Chinese)
}
