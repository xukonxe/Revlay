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
	AppShortDesc    string
	AppLongDesc     string
	AppVersion      string
	ConfigFileFlag  string
	LanguageFlag    string
	
	// Init Command
	InitShortDesc   string
	InitLongDesc    string
	InitSuccess     string
	InitFailed      string
	InitNameFlag    string
	InitHostFlag    string
	InitUserFlag    string
	InitPathFlag    string
	InitPromptName  string
	InitPromptHost  string
	InitPromptUser  string
	InitPromptPath  string
	
	// Deploy Command
	DeployShortDesc      string
	DeployLongDesc       string
	DeployStarting       string
	DeployDryRunMode     string
	DeploySSHTest        string
	DeploySSHSuccess     string
	DeployInProgress     string
	DeploySuccess        string
	DeployFailed         string
	DeployDryRunFlag     string
	DeployReleaseLive    string
	
	// Rollback Command
	RollbackShortDesc    string
	RollbackLongDesc     string
	RollbackSuccess      string
	RollbackFailed       string
	RollbackToRelease    string
	RollbackNoReleases   string
	
	// Releases Command
	ReleasesShortDesc    string
	ReleasesLongDesc     string
	ReleasesListHeader   string
	ReleasesNoReleases   string
	ReleasesCurrent      string
	
	// Status Command
	StatusShortDesc      string
	StatusLongDesc       string
	StatusCurrentRelease string
	StatusNoRelease      string
	StatusAppName        string
	StatusDeployPath     string
	StatusServerInfo     string
	
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
	DryRunSharedPaths    string
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
	DeploymentMode       string
	ZeroDowntime         string
	ShortDowntime        string
	DeploymentModeDesc   string
	
	// Service Management
	ServiceManagement    string
	ServicePort          string
	ServiceCommand       string
	ServiceHealthCheck   string
	ServiceRestartDelay  string
}

var currentLanguage Language = Chinese
var messages Messages

// Chinese messages
var chineseMessages = Messages{
	AppShortDesc:    "现代化、快速、无依赖的部署工具",
	AppLongDesc:     `Revlay是一个现代化的部署工具，提供原子部署、零停机部署和传统服务器部署的轻松回滚功能。\n\n它使用结构化的目录布局，包含releases、shared文件和原子符号链接切换，确保可靠的部署。`,
	AppVersion:      "版本",
	ConfigFileFlag:  "配置文件 (默认为 revlay.yml)",
	LanguageFlag:    "语言设置 (zh|en)",
	
	InitShortDesc:   "初始化新项目",
	InitLongDesc:    "初始化一个新的Revlay项目，创建默认的配置文件revlay.yml。",
	InitSuccess:     "✓ 项目初始化成功，配置文件已生成：%s",
	InitFailed:      "初始化失败：%v",
	InitNameFlag:    "应用名称",
	InitHostFlag:    "服务器主机名",
	InitUserFlag:    "SSH用户名",
	InitPathFlag:    "部署路径",
	InitPromptName:  "应用名称",
	InitPromptHost:  "服务器主机名",
	InitPromptUser:  "SSH用户名",
	InitPromptPath:  "部署路径",
	
	DeployShortDesc:      "部署新版本",
	DeployLongDesc:       "部署新版本到服务器。\n\n如果没有提供版本名称，将自动生成基于时间戳的名称。\n此命令将创建新的版本目录，链接共享路径，并切换current符号链接到新版本。",
	DeployStarting:       "🚀 开始部署版本：%s",
	DeployDryRunMode:     "🔍 预览模式 - 不会进行实际更改",
	DeploySSHTest:        "🔗 测试SSH连接...",
	DeploySSHSuccess:     "✓ SSH连接成功",
	DeployInProgress:     "📦 正在部署版本...",
	DeploySuccess:        "✓ 部署成功完成",
	DeployFailed:         "部署失败：%v",
	DeployDryRunFlag:     "显示将要执行的操作，但不实际部署",
	DeployReleaseLive:    "✓ 版本 %s 已在 %s 上线",
	
	RollbackShortDesc:    "回滚到指定版本",
	RollbackLongDesc:     "回滚到指定版本。如果没有指定版本，将回滚到上一个版本。",
	RollbackSuccess:      "✓ 成功回滚到版本：%s",
	RollbackFailed:       "回滚失败：%v",
	RollbackToRelease:    "🔄 回滚到版本：%s",
	RollbackNoReleases:   "没有找到可回滚的版本",
	
	ReleasesShortDesc:    "列出所有版本",
	ReleasesLongDesc:     "列出所有已部署的版本，显示版本名称、时间戳和当前激活状态。",
	ReleasesListHeader:   "📋 已部署的版本：",
	ReleasesNoReleases:   "没有找到已部署的版本",
	ReleasesCurrent:      " (当前)",
	
	StatusShortDesc:      "显示部署状态",
	StatusLongDesc:       "显示当前部署状态，包括激活的版本、应用信息和服务器配置。",
	StatusCurrentRelease: "当前版本：%s",
	StatusNoRelease:      "没有激活的版本",
	StatusAppName:        "应用名称：%s",
	StatusDeployPath:     "部署路径：%s",
	StatusServerInfo:     "服务器：%s@%s:%d",
	
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
	DryRunSharedPaths:    "🔗 将要链接的共享路径：",
	DryRunHooks:          "🪝 将要执行的钩子：",
	DryRunPreDeploy:      "部署前",
	DryRunPostDeploy:     "部署后",
	DryRunKeepReleases:   "🧹 保留 %d 个版本（旧版本将被清理）",
	
	ErrorConfigNotFound:  "配置文件 %s 未找到，请先运行 'revlay init'",
	ErrorConfigLoad:      "加载配置失败：%v",
	ErrorSSHConnect:      "连接服务器失败：%v",
	ErrorSSHTest:         "SSH连接测试失败：%v",
	ErrorDeployment:      "部署失败：%v",
	ErrorRollback:        "回滚失败：%v",
	ErrorNoReleases:      "没有找到可用的版本",
	ErrorReleaseNotFound: "版本 %s 不存在",
	
	DeploymentMode:       "部署模式",
	ZeroDowntime:         "零停机部署",
	ShortDowntime:        "短停机部署",
	DeploymentModeDesc:   "部署模式说明",
	
	ServiceManagement:    "服务管理",
	ServicePort:          "服务端口",
	ServiceCommand:       "服务命令",
	ServiceHealthCheck:   "健康检查",
	ServiceRestartDelay:  "重启延迟",
}

// English messages
var englishMessages = Messages{
	AppShortDesc:    "A modern, fast, dependency-free deployment tool",
	AppLongDesc:     `Revlay is a modern deployment tool that provides atomic deployments,\nzero-downtime deployments, and easy rollbacks for traditional server deployments.\n\nIt uses a structured directory layout with releases, shared files, and atomic\nsymlink switching to ensure reliable deployments.`,
	AppVersion:      "Version",
	ConfigFileFlag:  "config file (default is revlay.yml)",
	LanguageFlag:    "language setting (zh|en)",
	
	InitShortDesc:   "Initialize a new project",
	InitLongDesc:    "Initialize a new Revlay project and create a default revlay.yml configuration file.",
	InitSuccess:     "✓ Project initialized successfully, config file created: %s",
	InitFailed:      "Initialization failed: %v",
	InitNameFlag:    "Application name",
	InitHostFlag:    "Server hostname",
	InitUserFlag:    "SSH username",
	InitPathFlag:    "Deployment path",
	InitPromptName:  "Application name",
	InitPromptHost:  "Server hostname",
	InitPromptUser:  "SSH username",
	InitPromptPath:  "Deployment path",
	
	DeployShortDesc:      "Deploy a new release",
	DeployLongDesc:       "Deploy a new release to the server.\n\nIf no release name is provided, a timestamp-based name will be generated.\nThis command will create a new release directory, link shared paths,\nand switch the current symlink to the new release.",
	DeployStarting:       "🚀 Starting deployment of release: %s",
	DeployDryRunMode:     "🔍 DRY RUN MODE - No actual changes will be made",
	DeploySSHTest:        "🔗 Testing SSH connection...",
	DeploySSHSuccess:     "✓ SSH connection successful",
	DeployInProgress:     "📦 Deploying release...",
	DeploySuccess:        "✓ Deployment completed successfully",
	DeployFailed:         "Deployment failed: %v",
	DeployDryRunFlag:     "Show what would be done without actually deploying",
	DeployReleaseLive:    "✓ Release %s is now live at %s",
	
	RollbackShortDesc:    "Rollback to a specific release",
	RollbackLongDesc:     "Rollback to a specific release. If no release is specified, rollback to the previous release.",
	RollbackSuccess:      "✓ Successfully rolled back to release: %s",
	RollbackFailed:       "Rollback failed: %v",
	RollbackToRelease:    "🔄 Rolling back to release: %s",
	RollbackNoReleases:   "No releases found to rollback to",
	
	ReleasesShortDesc:    "List all releases",
	ReleasesLongDesc:     "List all deployed releases, showing release names, timestamps, and current active status.",
	ReleasesListHeader:   "📋 Deployed releases:",
	ReleasesNoReleases:   "No deployed releases found",
	ReleasesCurrent:      " (current)",
	
	StatusShortDesc:      "Show deployment status",
	StatusLongDesc:       "Show current deployment status including active release, application info, and server configuration.",
	StatusCurrentRelease: "Current release: %s",
	StatusNoRelease:      "No active release",
	StatusAppName:        "Application: %s",
	StatusDeployPath:     "Deploy path: %s",
	StatusServerInfo:     "Server: %s@%s:%d",
	
	DryRunPlan:           "📋 Deployment plan:",
	DryRunApplication:    "Application",
	DryRunServer:         "Server",
	DryRunRelease:        "Release",
	DryRunDeployPath:     "Deploy path",
	DryRunReleasesPath:   "Releases path",
	DryRunSharedPath:     "Shared path",
	DryRunCurrentPath:    "Current path",
	DryRunReleasePathFmt: "Release path",
	DryRunDirStructure:   "📂 Directory structure to be created:",
	DryRunSharedPaths:    "🔗 Shared paths to be linked:",
	DryRunHooks:          "🪝 Hooks to be executed:",
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
	
	DeploymentMode:       "Deployment Mode",
	ZeroDowntime:         "Zero Downtime Deployment",
	ShortDowntime:        "Short Downtime Deployment",
	DeploymentModeDesc:   "Deployment Mode Description",
	
	ServiceManagement:    "Service Management",
	ServicePort:          "Service Port",
	ServiceCommand:       "Service Command",
	ServiceHealthCheck:   "Health Check",
	ServiceRestartDelay:  "Restart Delay",
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