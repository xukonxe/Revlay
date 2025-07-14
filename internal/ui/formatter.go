package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// DeploymentFormatter 是部署过程的格式化程序
type DeploymentFormatter struct {
	releaseName string
	deployMode  string
	startTime   time.Time
	steps       int
	currentStep int
	progressbar *pterm.ProgressbarPrinter
	spinners    map[int]*pterm.SpinnerPrinter
	isEnabled   bool
}

// NewDeploymentFormatter 创建一个新的部署格式化程序
func NewDeploymentFormatter(releaseName, deployMode string, totalSteps int, enabled bool) *DeploymentFormatter {
	// 如果禁用了 UI，返回一个无操作的格式化程序
	if !enabled {
		return &DeploymentFormatter{
			isEnabled: false,
		}
	}

	// 创建进度条
	progressbar, _ := pterm.DefaultProgressbar.
		WithTotal(totalSteps).
		WithTitle("部署进度").
		WithRemoveWhenDone(true).
		Start()

	return &DeploymentFormatter{
		releaseName: releaseName,
		deployMode:  deployMode,
		startTime:   time.Now(),
		steps:       totalSteps,
		spinners:    make(map[int]*pterm.SpinnerPrinter),
		progressbar: progressbar,
		isEnabled:   true,
	}
}

// PrintBanner 打印部署横幅
func (f *DeploymentFormatter) PrintBanner() {
	if !f.isEnabled {
		return
	}

	// 打印横幅
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithMargin(2).
		Println("Revlay 部署系统")

	// 打印部署信息
	pterm.DefaultSection.Println("部署信息")
	pterm.Printf("%s: %s\n", pterm.LightCyan("版本"), pterm.LightGreen(f.releaseName))
	pterm.Printf("%s: %s\n", pterm.LightCyan("模式"), pterm.LightGreen(f.deployMode))
	pterm.Printf("%s: %s\n", pterm.LightCyan("开始时间"), pterm.LightGreen(f.startTime.Format("2006-01-02 15:04:05")))

	pterm.Println() // 空行
}

// StartStep 开始一个步骤
func (f *DeploymentFormatter) StartStep(step int, name string) {
	if !f.isEnabled {
		// 如果 UI 禁用，使用简单输出
		stepText := getChineseNumber(step + 1)
		fmt.Printf("步骤 %s: %s\n", stepText, name)
		return
	}

	f.currentStep = step

	// 更新进度条
	if f.progressbar != nil {
		f.progressbar.Current = step
		f.progressbar.UpdateTitle(fmt.Sprintf("部署进度 [%d/%d]", step+1, f.steps))
	}

	// 创建中文步骤号
	stepNumber := getChineseNumber(step + 1)

	// 创建带有步骤指示的 spinner
	stepPrefix := fmt.Sprintf("[%d/%d] 步骤%s: ",
		step+1, f.steps, stepNumber)

	// 创建并启动 spinner
	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(false).
		WithText(name).
		WithStyle(pterm.NewStyle(pterm.FgLightCyan)).
		Start(stepPrefix)

	// 存储 spinner 以便稍后更新
	f.spinners[step] = spinner
}

// CompleteStep 完成一个步骤
func (f *DeploymentFormatter) CompleteStep(step int, success bool, message string) {
	if !f.isEnabled {
		// 如果 UI 禁用，使用简单输出
		if success {
			fmt.Printf("  ✓ %s\n", message)
		} else {
			fmt.Printf("  ✗ %s\n", message)
		}
		return
	}

	// 获取对应的 spinner
	spinner := f.spinners[step]
	if spinner == nil {
		return
	}

	// 根据结果停止 spinner
	if success {
		spinner.Success(message)
	} else {
		spinner.Fail(message)
	}

	// 增加进度条
	if f.progressbar != nil {
		f.progressbar.Increment()
	}
}

// WarningStep 显示警告
func (f *DeploymentFormatter) WarningStep(step int, message string) {
	if !f.isEnabled {
		fmt.Printf("  ! %s\n", message)
		return
	}

	// 获取对应的 spinner
	spinner := f.spinners[step]
	if spinner == nil {
		return
	}

	// 显示警告
	spinner.Warning(message)
}

// PrintSystemOutput 打印系统输出，使其与 UI 区分开
func (f *DeploymentFormatter) PrintSystemOutput(message string) {
	if !f.isEnabled {
		fmt.Println(message)
		return
	}

	// 添加前缀区分系统输出
	pterm.FgGray.Println("│ SYSTEM │ " + message)
}

// CompleteDeployment 完成部署
func (f *DeploymentFormatter) CompleteDeployment(success bool, message string) {
	if !f.isEnabled {
		if success {
			fmt.Println("✓ 部署完成")
		} else {
			fmt.Printf("✗ 部署失败: %s\n", message)
		}
		return
	}

	// 显示最终结果
	pterm.Println() // 空行

	// 计算总时长
	duration := time.Since(f.startTime).Round(time.Second)

	// 创建结果区域
	if success {
		pterm.Success.Println("部署完成")
		pterm.Printf("  版本: %s\n", pterm.Green(f.releaseName))
		pterm.Printf("  用时: %s\n", pterm.Green(duration))
	} else {
		pterm.Error.Println("部署失败")
		pterm.Printf("  错误: %s\n", pterm.Red(message))
		pterm.Printf("  版本: %s\n", pterm.Red(f.releaseName))
		pterm.Printf("  用时: %s\n", pterm.Red(duration))
	}

	// 清理
	if f.progressbar != nil {
		f.progressbar.Stop()
	}
}

// LogSystemMessage 记录系统消息
func (f *DeploymentFormatter) LogSystemMessage(message string) {
	if !f.isEnabled {
		fmt.Println(message)
		return
	}

	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// 以灰色显示系统消息
		pterm.FgGray.Println("│ " + line)
	}
}

// getChineseNumber 将数字转换为中文数字
func getChineseNumber(num int) string {
	if num <= 0 {
		return ""
	}
	chineseNumbers := []string{"一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}
	if num <= 10 {
		return chineseNumbers[num-1]
	}
	if num < 20 {
		return "十" + chineseNumbers[num-11]
	}
	return fmt.Sprintf("%d", num)
}
