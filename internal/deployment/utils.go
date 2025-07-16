package deployment

import (
	"fmt"
	"strconv"

	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ui"
)

// stepLogger helps in logging numbered steps.
type stepLogger struct {
	step      int
	formatter *ui.DeploymentFormatter
	useUI     bool
}

func newStepLogger() *stepLogger {
	return &stepLogger{step: 0}
}

// 创建支持 UI 的 stepLogger
func newFormattedStepLogger(formatter *ui.DeploymentFormatter) *stepLogger {
	return &stepLogger{
		step:      0,
		formatter: formatter,
		useUI:     formatter != nil,
	}
}

func (l *stepLogger) Print(message string) {
	l.step++

	if l.useUI && l.formatter != nil {
		// 使用格式化程序显示步骤
		l.formatter.StartStep(l.step-1, message)
	} else {
		// 原始的输出逻辑
		var stepText string
		if i18n.GetLanguage() == i18n.Chinese {
			stepText = i18n.Sprintf(i18n.T().DeployStep, convertToChineseNumber(l.step))
		} else {
			stepText = i18n.Sprintf(i18n.T().DeployStep, l.step)
		}

		// 格式化并打印消息
		fmt.Println(color.Cyan(fmt.Sprintf("%s: %s", stepText, message)))
	}
}

func (l *stepLogger) Success(message string) {
	if l.useUI && l.formatter != nil {
		l.formatter.StepSuccess(l.step, message)
	} else {
		fmt.Println(color.Green("  ✓ " + message))
	}
}

func (l *stepLogger) Warn(message string) {
	if l.useUI && l.formatter != nil {
		l.formatter.StepWarn(l.step, message)
	} else {
		fmt.Println(color.Yellow("  ! " + message))
	}
}

func (l *stepLogger) Error(message string) {
	if l.useUI && l.formatter != nil {
		l.formatter.StepWarn(l.step, message)
	} else {
		fmt.Println(color.Red("  ✗ " + message))
	}
}

// SystemLog 记录系统日志消息
func (l *stepLogger) SystemLog(message string) {
	if l.useUI && l.formatter != nil {
		l.formatter.StepLog(l.step, message)
	} else {
		fmt.Println(message)
	}
}

// convertToChineseNumber converts an integer to a Chinese number string, matching original logic.
func convertToChineseNumber(num int) string {
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
	return strconv.Itoa(num)
}
