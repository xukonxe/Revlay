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
	InitDirectoryFlag string
	InitPromptName    string
	InitPromptPath    string
	InitForceFlag     string

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
	DeployFromDirFlag string

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
	StatusShortDesc        string
	StatusLongDesc         string
	StatusCurrentRelease   string
	StatusNoRelease        string
	StatusAppName          string
	StatusDeployPath       string
	StatusServerInfo       string
	StatusActive           string
	StatusDirectoryDetails string
	StatusDirFailed        string

	// Service Command
	ServiceShortDesc          string
	ServiceLongDesc           string
	ServiceStartShortDesc     string
	ServiceStartLongDesc      string
	ServiceStarting           string
	ServiceStartSuccess       string
	ServiceStartFailed        string
	ServiceStartNotConfigured string
	ServiceStopShortDesc      string
	ServiceStopLongDesc       string
	ServiceStopping           string
	ServiceStopSuccess        string
	ServiceStopFailed         string
	ServiceStopNotConfigured  string
	ServiceStopNotRunning     string
	ServiceNotFound           string
	ServiceIdRequired         string
	ServiceNoReleaseFound     string
	ServiceNotConfigured      string
	ServiceAlreadyRunning     string
	ServiceStalePidFile       string

	// Push Command
	PreflightCheckFailed string
	PushShortDesc        string
	PushLongDesc         string
	PushStarting         string
	PushCheckingRemote   string
	PushRemoteFound      string
	PushAppFound         string
	PushCreatingTempDir  string
	PushTempDirCreated   string
	PushCleaningUp       string
	PushCleanupFailed    string
	PushCleanupComplete  string
	PushSyncingFiles     string
	PushSyncComplete     string
	PushTriggeringDeploy string
	PushComplete         string

	// Deployment Steps
	DeployPreflightChecks             string
	DeployLockError                   string
	DeployAlreadyInProgress           string
	DeployCreatingDir                 string
	DeployDirCreationError            string
	DeploySetupDirs                   string
	DeployEnsuringDir                 string
	DeployPopulatingDir               string
	DeployCopyingContent              string
	DeployMovingContent               string
	DeployRenameFailed                string
	DeployCreatedEmpty                string
	DeployEmptyNote                   string
	DeployLinkingShared               string
	DeployLinking                     string
	DeployPreHooks                    string
	DeployActivating                  string
	DeployPointingSymlink             string
	DeployStoppingService             string
	DeployStopServiceFailed           string
	DeployStartingService             string
	DeployStartServiceFailed          string
	DeployRestartingService           string
	DeployHealthCheck                 string
	DeployHealthAttempt               string
	DeployHealthFailed                string
	DeployHealthPassed                string
	DeployPostHooks                   string
	DeployPruning                     string
	DeployPruningRelease              string
	DeployPruningLogFile              string
	DeployPruningLogFileFailed        string
	DeployCmdExecFailed               string
	DeployZeroDowntimeWarning         string
	DeployRollbackStart               string
	DeployRollbackSuccess             string
	DeployNoReleasesFound             string
	DeployExecZeroDowntime            string
	DeployExecShortDowntime           string
	DeployStep                        string
	DeployDeterminePorts              string
	DeployStartNewRelease             string
	DeployHealthCheckOnPort           string
	DeploySwitchProxy                 string
	DeployActivateSymlink             string
	DeployStopOldService              string
	DeployErrProcExitedEarly          string
	DeployErrProcExitedEarlyWithError string
	DeployCurrentPortInfo             string
	DeployNewPortInfo                 string
	DeployDeterminePortsWarn          string
	DeployDeterminePortsSuccess       string
	DeployStartNewReleaseFailed       string
	DeployStartNewReleaseSuccess      string
	DeploySwitchProxySuccess          string
	DeployStopOldServiceWarn          string
	DeployStopOldServiceSuccess       string
	DeployFindOldPidFailed            string
	DeployCleanup                     string
	DeployCleanupFailed               string
	DeployCleanupSuccess              string
	DeployVersion                     string
	DeployMode                        string
	DeployModeShort                   string
	DeployModeZero                    string
	DeployStartTime                   string
	DeployStart                       string
	Deploying                         string
	DeployCurrentSymlink              string
	DeploySymlinkTo                   string
	DeployCurrentRelease              string
	DeployRetain                      string
	DeployClean                       string
	DeployCleanWarn                   string
	DeployCleanSuccess                string
	DeployFillRelease                 string
	DeployCopy                        string
	DeployCopyFailed                  string
	DeployLinkShared                  string
	DeployLinkSharedFailed            string
	DeployStopService                 string
	DeploySetupDirsSuccess            string
	DeployPruningWarn                 string
	DeployPruningSuccess              string
	DeployOldPidNotFound              string
	DeployFindOldProcessFailed        string
	DeployStopOldProcessFailed        string

	// SSH Messages
	SSHRunningRemote string
	SSHCommandFailed string
	SSHStreamFailed  string
	SSHRsyncCommand  string
	SSHRsyncFailed   string

	// Agent Messages
	AgentRunning string

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

	ServiceGracefulShutdown string
	ServiceStartInitiated   string
}

var currentLanguage Language = Chinese
var messages Messages

// Chinese messages
var chineseMessages = Messages{
	AppShortDesc:   "一个现代、快速、零依赖的部署和服务器生命周期管理工具。",
	AppLongDesc:    `Revlay是一个用于部署和管理Web应用程序的命令行工具。`,
	ConfigFileFlag: "配置文件路径 (默认为revlay.yml)",
	LanguageFlag:   "输出语言 (例如: 'en', 'zh')",

	// init command
	InitShortDesc:     "用revlay.yml文件初始化新项目",
	InitLongDesc:      `init命令在当前或指定目录中创建新的revlay.yml配置文件。`,
	InitNameFlag:      "应用名称",
	InitPathFlag:      "服务器上的部署路径",
	InitDirectoryFlag: "初始化的目标目录",
	InitPromptName:    "应用名称",
	InitPromptPath:    "部署路径",
	InitFailed:        "初始化失败: %v",
	InitSuccess:       "配置文件已创建于 %s",
	InitForceFlag:     "覆盖现有的revlay.yml文件（如果存在）",

	// deploy command
	DeployShortDesc:   "将应用程序部署到服务器",
	DeployLongDesc:    "向服务器部署新的版本。\n\n如果未提供版本名称，将生成基于时间戳的名称。\n该命令将创建新的版本目录，链接共享路径，\n并将当前符号链接切换到新版本。",
	DeployStarting:    "🚀 开始部署版本：%s",
	DeployDryRunMode:  "🔍 演示模式 - 不会进行实际更改",
	DeploySSHTest:     "🔗 测试SSH连接...",
	DeploySSHSuccess:  "✓ SSH连接成功",
	DeployInProgress:  "📦 正在部署版本...",
	DeploySuccess:     "✓ 部署成功完成",
	DeployFailed:      "部署失败: %v",
	DeployDryRunFlag:  "显示部署过程但不实际执行",
	DeployReleaseLive: "✓ 版本 %s 现已在 %s 上线",
	DeployDryRunPlan:  "部署计划:",
	DeployFromDirFlag: "从特定目录部署而不是从空目录",

	// releases command
	ReleasesShortDesc:  "列出所有已部署的版本",
	ReleasesLongDesc:   "列出在版本目录中找到的所有版本。",
	ReleasesListHeader: "📋 已部署的版本:",
	ReleasesNoReleases: "未找到任何版本。",
	ReleasesCurrent:    " (当前)",
	ReleasesHeader:     "%-18s %s",
	ErrorReleasesList:  "列出版本失败: %v",

	// rollback command
	RollbackShortDesc:  "回滚到之前的版本",
	RollbackLongDesc:   "通过切换'current'符号链接，将应用程序回滚到指定的版本。",
	RollbackStarting:   "正在回滚到版本 %s...",
	RollbackSuccess:    "成功回滚到 %s。",
	RollbackFailed:     "回滚失败: %v",
	RollbackToRelease:  "🔄 正在回滚到版本: %s",
	RollbackNoReleases: "未找到可回滚的版本",

	// Status Command
	StatusShortDesc:        "显示部署状态",
	StatusLongDesc:         "显示当前部署的版本和其他状态信息。",
	StatusCurrentRelease:   "当前版本: %s",
	StatusNoRelease:        "没有活动的版本",
	StatusAppName:          "应用: %s",
	StatusDeployPath:       "部署路径: %s",
	StatusServerInfo:       "服务器: %s@%s:%d",
	StatusActive:           "活动",
	StatusDirectoryDetails: "目录详情:",
	StatusDirFailed:        "  - 无法获取目录详情: %v",

	// Service Command
	ServiceShortDesc:          "管理 Revlay 服务",
	ServiceLongDesc:           "管理 Revlay 服务列表，包括添加、删除和列出服务。",
	ServiceStartShortDesc:     "启动一个服务",
	ServiceStartLongDesc:      "启动全局服务列表中的指定服务。",
	ServiceStarting:           "正在启动服务 '%s'...",
	ServiceStartSuccess:       "✅ 服务 '%s' 已成功启动，进程ID: %d。",
	ServiceStartFailed:        "❌ 启动服务 '%s' 失败: %v",
	ServiceStartNotConfigured: "❌ 服务 '%s' 没有配置启动命令，无法启动。",
	ServiceStopShortDesc:      "停止一个服务",
	ServiceStopLongDesc:       "停止全局服务列表中的指定服务。",
	ServiceStopping:           "正在停止服务 '%s'...",
	ServiceStopSuccess:        "✅ 服务 '%s' 已成功停止。",
	ServiceStopFailed:         "❌ 停止服务 '%s' 失败: %v",
	ServiceStopNotConfigured:  "❌ 服务 '%s' 没有配置停止命令，无法停止。",
	ServiceStopNotRunning:     "⚠️ 服务 '%s' 未运行。",
	ServiceNotFound:           "❌ 未找到服务 '%s'。",
	ServiceIdRequired:         "请指定服务 ID。",
	ServiceNoReleaseFound:     "❌ 服务 '%s' 未部署任何版本。",
	ServiceNotConfigured:      "❌ 服务 '%s' 配置不完整，无法执行操作。",
	ServiceAlreadyRunning:     "⚠️ 服务 '%s' 已在运行，进程ID: %d。",
	ServiceStalePidFile:       "发现过时的PID文件，启动前将自动删除。",

	// Push Command
	PreflightCheckFailed: "Pre-flight check failed: command '%s' not found. Please install it and ensure it's in your PATH. Error: %v",
	PushShortDesc:        "推送本地目录到远程并部署",
	PushLongDesc:         `此命令使用rsync将本地目录推送到远程服务器，然后触发远程机器上的'revlay deploy'命令。\n\n它通过在一个步骤中打包、传输和激活新版本，简化了部署过程。`,
	PushStarting:         "🚀 开始推送到 %s 的应用 '%s'...",
	PushCheckingRemote:   "🔎 检查远程环境...",
	PushRemoteFound:      "✅ 找到远程'revlay'命令。",
	PushAppFound:         "✅ 找到远程应用 '%s'。",
	PushCreatingTempDir:  "📁 在远程创建临时目录...",
	PushTempDirCreated:   "✅ 已创建临时目录: %s",
	PushCleaningUp:       "\n🧹 清理远程临时目录...",
	PushCleanupFailed:    "⚠️ 清理临时目录 %s 失败: %v",
	PushCleanupComplete:  "✅ 清理完成。",
	PushSyncingFiles:     "🚚 同步文件到 %s...",
	PushSyncComplete:     "✅ 文件同步成功完成。",
	PushTriggeringDeploy: "🚢 触发远程部署应用 '%s'...",
	PushComplete:         "\n🎉 推送和部署成功完成!",

	// Deployment Steps
	DeployPreflightChecks:             "执行预检...",
	DeployLockError:                   "获取部署锁失败: %v",
	DeployAlreadyInProgress:           "另一个部署似乎正在进行中（锁文件存在）。如果不是这样，请手动删除'revlay.lock'。",
	DeployCreatingDir:                 "  - 创建目录: %s",
	DeployDirCreationError:            "创建目录 %s 失败: %v",
	DeploySetupDirs:                   "设置目录...",
	DeployEnsuringDir:                 "  - 确保目录存在: %s",
	DeployPopulatingDir:               "填充版本目录...",
	DeployCopyingContent:              "  - 从 %s 复制内容",
	DeployMovingContent:               "  - 从 %s 移动内容",
	DeployRenameFailed:                "  - 重命名失败，回退到复制...",
	DeployCreatedEmpty:                "  - 创建空版本目录: %s",
	DeployEmptyNote:                   "  - 注意: 未指定源目录。使用部署前钩子填充此目录。",
	DeployLinkingShared:               "链接共享路径...",
	DeployLinking:                     "  - 链接: %s -> %s",
	DeployPreHooks:                    "执行部署前钩子...",
	DeployActivating:                  "激活新版本...",
	DeployPointingSymlink:             "  - 将'current'符号链接指向: %s",
	DeployStoppingService:             "停止当前服务...",
	DeployStopServiceFailed:           "警告：停止旧服务失败：%v。可能没有服务在运行。",
	DeployStartingService:             "启动新服务...",
	DeployStartServiceFailed:          "启动新服务失败：%v",
	DeployRestartingService:           "重启服务...",
	DeployHealthCheck:                 "执行健康检查...",
	DeployHealthAttempt:               "  - 健康检查尝试 #%d 对 %s...",
	DeployHealthFailed:                " ✗",
	DeployHealthPassed:                " ✓",
	DeployPostHooks:                   "执行部署后钩子...",
	DeployPruning:                     "清理旧版本...",
	DeployPruningRelease:              "清理旧版本: %s",
	DeployPruningLogFile:              "清理日志文件: %s",
	DeployPruningLogFileFailed:        "清理日志文件 %s 失败: %v",
	DeployCmdExecFailed:               "命令执行失败: %s\n%s",
	DeployZeroDowntimeWarning:         "警告: 零停机部署目前是简化版，行为与标准部署相同。",
	DeployRollbackStart:               "正在回滚到版本 %s...",
	DeployRollbackSuccess:             "回滚成功。",
	DeployNoReleasesFound:             "未找到任何版本。",
	DeployExecZeroDowntime:            "零停机部署模式",
	DeployExecShortDowntime:           "执行短停机部署...",
	DeployStep:                        "# 步骤 %s: %s",
	DeployDeterminePorts:              "确定新旧服务端口",
	DeployStartNewRelease:             "在端口 %d 上启动新版本",
	DeployHealthCheckOnPort:           "在端口 %d 上执行健康检查",
	DeploySwitchProxy:                 "健康检查通过。切换代理流量到端口 %d...",
	DeployActivateSymlink:             "激活新版本符号链接...",
	DeployStopOldService:              "在端口 %d 上停止旧服务 (等待 %s)...",
	DeployErrProcExitedEarly:          "新版本进程在健康检查完成前已正常退出（状态码0），服务应保持在线状态",
	DeployErrProcExitedEarlyWithError: "新版本进程在启动期间意外退出：%v",
	DeployCurrentPortInfo:             "  - 当前服务运行于端口: %d",
	DeployNewPortInfo:                 "  - 新服务将启动于端口: %d",
	DeployDeterminePortsWarn:          "无法确定当前端口: %v。将使用默认主端口。",
	DeployDeterminePortsSuccess:       "端口确定完成",
	DeployStartNewReleaseFailed:       "启动新版本服务失败: %v",
	DeployStartNewReleaseSuccess:      "新版本服务已启动",
	DeploySwitchProxySuccess:          "代理流量切换成功。",
	DeployStopOldServiceWarn:          "警告: 旧服务 %s 可能没有在端口 %d 上运行。",
	DeployStopOldServiceSuccess:       "旧服务 %s 已成功停止。",
	DeployFindOldPidFailed:            "找不到旧服务进程ID: %v",
	DeployCleanup:                     "清理旧版本...",
	DeployCleanupFailed:               "清理失败: %v",
	DeployCleanupSuccess:              "清理完成。",
	DeployVersion:                     "版本: %s",
	DeployMode:                        "部署模式: %s",
	DeployModeShort:                   "短停机部署",
	DeployModeZero:                    "零停机部署",
	DeployStartTime:                   "部署开始时间: %s",
	DeployStart:                       "部署中...",
	Deploying:                         "部署中...",
	DeployCurrentSymlink:              "当前符号链接: %s",
	DeploySymlinkTo:                   "将符号链接指向: %s",
	DeployCurrentRelease:              "当前版本: %s",
	DeployRetain:                      "保留旧版本",
	DeployClean:                       "清理旧版本",
	DeployCleanWarn:                   "警告: 清理旧版本可能导致服务中断。",
	DeployCleanSuccess:                "清理完成。",
	DeployFillRelease:                 "填充新版本目录",
	DeployCopy:                        "复制内容",
	DeployCopyFailed:                  "复制失败: %v",
	DeployLinkShared:                  "链接共享路径",
	DeployLinkSharedFailed:            "链接共享路径失败: %v",
	DeployStopService:                 "停止服务",
	DeploySetupDirsSuccess:            "设置目录成功。",
	DeployPruningWarn:                 "清理旧版本时发出警告: %v",
	DeployPruningSuccess:              "成功清理旧版本。",
	DeployOldPidNotFound:              "未找到旧服务的PID。",
	DeployFindOldProcessFailed:        "通过PID %d 查找旧进程失败: %v",
	DeployStopOldProcessFailed:        "停止旧进程 %d 失败: %v",

	// SSH Messages
	SSHRunningRemote: "在远程服务器上运行: %s",
	SSHCommandFailed: "远程命令执行失败: %v",
	SSHStreamFailed:  "ssh流命令失败: %w",
	SSHRsyncCommand:  "  -> 运行: rsync %s",
	SSHRsyncFailed:   "rsync命令失败: %w",

	// Agent Messages
	AgentRunning: "Revlay Agent 正在运行...",

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

	ServiceGracefulShutdown: "正在为进程 %d 请求平滑关闭...",
	ServiceStartInitiated:   "服务启动已初始化。PID: %d, 日志: %s",
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
	InitForceFlag:     "Overwrite existing revlay.yml if it exists",

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
	DeployFromDirFlag: "Deploy from a specific directory instead of an empty one",

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
	StatusShortDesc:        "Show the status of the deployment",
	StatusLongDesc:         "Displays the current deployed release and other status information.",
	StatusCurrentRelease:   "Current release: %s",
	StatusNoRelease:        "No active release",
	StatusAppName:          "Application: %s",
	StatusDeployPath:       "Deploy path: %s",
	StatusServerInfo:       "Server: %s@%s:%d",
	StatusActive:           "Active",
	StatusDirectoryDetails: "Directory Details:",
	StatusDirFailed:        "  - Could not get directory details: %v",

	// Service Command
	ServiceShortDesc:          "Manage Revlay services",
	ServiceLongDesc:           "Manage the Revlay services list, including adding, removing, and listing services.",
	ServiceStartShortDesc:     "Start a service",
	ServiceStartLongDesc:      "Start a service from the global services list.",
	ServiceStarting:           "Starting service '%s'...",
	ServiceStartSuccess:       "✅ Service '%s' started successfully with PID: %d.",
	ServiceStartFailed:        "❌ Failed to start service '%s': %v",
	ServiceStartNotConfigured: "❌ Service '%s' has no start command configured.",
	ServiceStopShortDesc:      "Stop a service",
	ServiceStopLongDesc:       "Stop a service from the global services list.",
	ServiceStopping:           "Stopping service '%s'...",
	ServiceStopSuccess:        "✅ Service '%s' stopped successfully.",
	ServiceStopFailed:         "❌ Failed to stop service '%s': %v",
	ServiceStopNotConfigured:  "❌ Service '%s' has no stop command configured.",
	ServiceStopNotRunning:     "⚠️ Service '%s' is not running.",
	ServiceNotFound:           "❌ Service '%s' not found.",
	ServiceIdRequired:         "Please specify a service ID.",
	ServiceNoReleaseFound:     "❌ No releases found for service '%s'.",
	ServiceNotConfigured:      "❌ Service '%s' is not properly configured.",
	ServiceAlreadyRunning:     "⚠️ Service '%s' is already running with PID: %d.",
	ServiceStalePidFile:       "Stale PID file found and removed.",

	// Push Command
	PreflightCheckFailed: "本地环境预检失败：命令 '%s' 未找到。请安装该命令并确保其位于系统的 PATH 环境变量中。错误: %v",
	PushShortDesc:        "Push and deploy an application to a remote server",
	PushLongDesc:         "Compresses a local directory, securely transfers it to a remote server using rsync, and then executes the 'deploy' command on the server to complete the deployment process.",
	PushStarting:         "🚀 开始推送到 '%s' (应用: '%s')...",
	PushCheckingRemote:   "🔎 Checking remote environment...",
	PushRemoteFound:      "✅ Remote 'revlay' command found. Version: %s",
	PushAppFound:         "✅ Found remote application '%s'.",
	PushCreatingTempDir:  "📁 Creating temporary directory on remote server...",
	PushTempDirCreated:   "✅ Temporary directory created at '%s'.",
	PushCleaningUp:       "🧹 Cleaning up temporary directory...",
	PushCleanupFailed:    "⚠️ Failed to clean up temporary directory %s: %v",
	PushCleanupComplete:  "✅ Cleanup complete.",
	PushSyncingFiles:     "🚚 Syncing files to %s...",
	PushSyncComplete:     "✅ File sync completed successfully.",
	PushTriggeringDeploy: "🚢 Triggering remote deployment for app '%s'...",
	PushComplete:         "\n🎉 Push and deploy completed successfully!",

	// Deployment Steps
	DeployPreflightChecks:             "Running pre-flight checks...",
	DeployLockError:                   "Failed to acquire deployment lock: %v",
	DeployAlreadyInProgress:           "Another deployment appears to be in progress (lock file exists). If this is not true, please remove 'revlay.lock' manually.",
	DeployCreatingDir:                 "  - Creating directory: %s",
	DeployDirCreationError:            "Failed to create directory %s: %v",
	DeploySetupDirs:                   "Setting up directories...",
	DeployEnsuringDir:                 "  - Ensuring directory exists: %s",
	DeployPopulatingDir:               "Populating release directory...",
	DeployCopyingContent:              "  - Copying content from %s",
	DeployMovingContent:               "  - Moving content from %s",
	DeployRenameFailed:                "  - Rename failed, falling back to copy...",
	DeployCreatedEmpty:                "  - Created empty release directory: %s",
	DeployEmptyNote:                   "  - Note: No source specified. Use pre_deploy hooks to populate this directory.",
	DeployLinkingShared:               "Step 3: Linking shared paths...",
	DeployLinking:                     "  - Linking: %s -> %s",
	DeployPreHooks:                    "Step 4: Running pre-deploy hooks...",
	DeployActivating:                  "Step 5: Activating new release...",
	DeployPointingSymlink:             "  - Pointing 'current' symlink to: %s",
	DeployStoppingService:             "Stopping current service...",
	DeployStopServiceFailed:           "Warning: failed to stop old service: %v. It may not have been running.",
	DeployStartingService:             "Starting new service...",
	DeployStartServiceFailed:          "Failed to start new service: %v",
	DeployRestartingService:           "Step 6: Restarting service...",
	DeployHealthCheck:                 "Step 7: Performing health check...",
	DeployHealthAttempt:               "  - Health check attempt #%d to %s...",
	DeployHealthFailed:                " Failed",
	DeployHealthPassed:                " Passed.",
	DeployPostHooks:                   "Step 8: Running post-deploy hooks...",
	DeployPruning:                     "Step 9: Pruning old releases...",
	DeployPruningRelease:              "Pruning old release: %s",
	DeployPruningLogFile:              "Pruning log file: %s",
	DeployPruningLogFileFailed:        "Failed to prune log file %s: %v",
	DeployCmdExecFailed:               "Command failed: %s\n%s",
	DeployZeroDowntimeWarning:         "Warning: Zero-downtime deployment is currently simplified and acts like a standard deploy.",
	DeployRollbackStart:               "Rolling back to release %s...",
	DeployRollbackSuccess:             "Rollback successful.",
	DeployNoReleasesFound:             "No releases found.",
	DeployExecZeroDowntime:            "Zero-Downtime Deployment",
	DeployExecShortDowntime:           "Executing short-downtime deployment...",
	DeployStep:                        "# Step %s: %s",
	DeployDeterminePorts:              "Determining ports for old and new services",
	DeployStartNewRelease:             "Starting new release on port %d",
	DeployHealthCheckOnPort:           "Performing health check on port %d",
	DeploySwitchProxy:                 "Health check passed. Switching proxy traffic to port %d...",
	DeployActivateSymlink:             "Activating new release symlink...",
	DeployStopOldService:              "Stopping old service on port %d (after %s grace period)...",
	DeployErrProcExitedEarly:          "new release process exited cleanly (status 0) before health check passed; a service is expected to stay online",
	DeployErrProcExitedEarlyWithError: "new release process exited unexpectedly during startup: %v",
	DeployCurrentPortInfo:             "  - Current service detected on port: %d",
	DeployNewPortInfo:                 "  - New service will start on port: %d",
	DeployDeterminePortsWarn:          "Could not determine current port: %v. Falling back to default primary port.",
	DeployDeterminePortsSuccess:       "Port determination complete",
	DeployStartNewReleaseFailed:       "Failed to start new release: %v",
	DeployStartNewReleaseSuccess:      "New release started successfully",
	DeploySwitchProxySuccess:          "Proxy traffic switched successfully.",
	DeployStopOldServiceWarn:          "Warning: old service %s may not be running on port %d.",
	DeployStopOldServiceSuccess:       "Old service %s stopped successfully.",
	DeployFindOldPidFailed:            "Failed to find old service PID: %v",
	DeployCleanup:                     "Cleaning up old releases...",
	DeployCleanupFailed:               "Cleanup failed: %v",
	DeployCleanupSuccess:              "Cleanup completed.",
	DeployVersion:                     "Version: %s",
	DeployMode:                        "Deployment Mode: %s",
	DeployModeShort:                   "Short Downtime Deployment",
	DeployModeZero:                    "Zero Downtime Deployment",
	DeployStartTime:                   "Deployment start time: %s",
	DeployStart:                       "Deploying...",
	Deploying:                         "Deploying...",
	DeployCurrentSymlink:              "Current symlink: %s",
	DeploySymlinkTo:                   "Pointing symlink to: %s",
	DeployCurrentRelease:              "Current release: %s",
	DeployRetain:                      "Retain old releases",
	DeployClean:                       "Clean up old releases",
	DeployCleanWarn:                   "Warning: cleaning up old releases may cause service interruption.",
	DeployCleanSuccess:                "Cleanup completed.",
	DeployFillRelease:                 "Fill new release directory",
	DeployCopy:                        "Copy content",
	DeployCopyFailed:                  "Copy failed: %v",
	DeployLinkShared:                  "Link shared paths",
	DeployLinkSharedFailed:            "Failed to link shared paths: %v",
	DeployStopService:                 "Stop service",
	DeploySetupDirsSuccess:            "Directories set up successfully.",
	DeployPruningWarn:                 "Warning during old release cleanup: %v",
	DeployPruningSuccess:              "Successfully cleaned up old releases.",
	DeployOldPidNotFound:              "Could not find PID for the old service.",
	DeployFindOldProcessFailed:        "Failed to find old process with PID %d: %v",
	DeployStopOldProcessFailed:        "Failed to stop old process %d: %v",

	// SSH Messages
	SSHRunningRemote: "Running on remote server: %s",
	SSHCommandFailed: "Remote command execution failed: %v",
	SSHStreamFailed:  "ssh stream command failed: %w",
	SSHRsyncCommand:  "  -> Running: rsync %s",
	SSHRsyncFailed:   "rsync command failed: %w",

	// Agent Messages
	AgentRunning: "Revlay Agent is running...",

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
	ServicePort:         "Port",
	ServiceCommand:      "Command",
	ServiceHealthCheck:  "Health Check",
	ServiceRestartDelay: "Restart Delay",

	ServiceGracefulShutdown: "Requesting graceful shutdown for process with PID %d...",
	ServiceStartInitiated:   "Service start initiated. PID: %d, Logs: %s",
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
