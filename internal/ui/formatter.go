package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
)

// DeploymentFormatter handles the presentation of deployment information.
type DeploymentFormatter struct {
	releaseName    string
	deploymentMode string
	totalSteps     int
	startTime      time.Time
	multiPrinter   pterm.MultiPrinter
	spinners       map[int]*pterm.SpinnerPrinter
	ptermArea      *pterm.AreaPrinter
	beautify       bool
	liveLogs       *strings.Builder
}

// NewDeploymentFormatter creates a new formatter for deployment UI.
func NewDeploymentFormatter(releaseName, deploymentMode string, totalSteps int, beautify bool) *DeploymentFormatter {
	f := &DeploymentFormatter{
		releaseName:    releaseName,
		deploymentMode: deploymentMode,
		totalSteps:     totalSteps,
		startTime:      time.Now(),
		spinners:       make(map[int]*pterm.SpinnerPrinter),
		beautify:       beautify,
		liveLogs:       new(strings.Builder),
	}
	if beautify {
		f.multiPrinter = pterm.DefaultMultiPrinter
		f.ptermArea, _ = pterm.DefaultArea.Start()
		f.multiPrinter.Start()
	}
	return f
}

// renderAsciiArt 生成 "Revlay" 的 ASCII Art 字符串。
func (f *DeploymentFormatter) renderAsciiArt() string {
	s, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("Revlay", pterm.NewStyle(pterm.FgCyan)),
	).Srender()
	return s
}

// banner 返回一个带有全宽背景的标题字符串。
// 调用此函数前应确保终端宽度足够。
func (f *DeploymentFormatter) banner() string {
	return pterm.DefaultHeader.
		WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgBlack)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Sprint(f.renderAsciiArt())
}

func (f *DeploymentFormatter) updateArea() {
	if !f.beautify || f.ptermArea == nil {
		return
	}
	f.ptermArea.Update(f.liveLogs.String())
}

func (f *DeploymentFormatter) StartStreaming(releaseName string) {
	if f.beautify {
		f.liveLogs.WriteString(pterm.DefaultSection.Sprintf("Streaming logs for %s...\n", releaseName))
		f.updateArea()
	}
}

func (f *DeploymentFormatter) StopStreaming() {
	if f.beautify {
		f.liveLogs.WriteString(pterm.DefaultSection.Sprintf("...stopped streaming logs.\n"))
		f.updateArea()
	}
}

func (f *DeploymentFormatter) StreamLog(releaseName, streamType, message string) {
	if f.beautify {
		prefix := fmt.Sprintf("[%s-%s]", releaseName, streamType)
		logLine := fmt.Sprintf("%s %s %s\n", time.Now().Format("15:04:05"), pterm.Gray(prefix), message)
		f.liveLogs.WriteString(logLine)
		f.updateArea()
	} else {
		prefix := fmt.Sprintf("[%s-%s]", releaseName, streamType)
		log.Printf("%s %s", pterm.Gray(prefix), message)
	}
}

// PrintBanner 检查终端宽度并打印合适的标题。
func (f *DeploymentFormatter) PrintBanner() {
	if f.beautify {
		// 获取终端宽度，如果失败则使用一个保守的默认值
		width, _, _ := pterm.GetTerminalSize()

		// 获取 ASCII Art 的实际宽度（取最长的一行）
		asciiArt := f.renderAsciiArt()
		asciiArtWidth := 0
		for _, line := range strings.Split(asciiArt, "\n") {
			if len(line) > asciiArtWidth {
				asciiArtWidth = len(line)
			}
		}

		// 如果终端宽度小于 ASCII Art 宽度加上一些边距，则进行降级处理
		if width < asciiArtWidth+10 {
			pterm.Println(asciiArt) // 只打印文字，不打印全宽背景
		} else {
			pterm.Println(f.banner()) // 宽度足够，打印酷炫的标题
		}

		pterm.Println(f.infoPanel())
	}
}

func (f *DeploymentFormatter) infoPanel() string {
	// ... (infoPanel generation logic - can be kept as is)
	panels := pterm.Panels{
		{{Data: pterm.DefaultBox.Sprint(f.deploymentInfo())}},
	}
	panel, _ := pterm.DefaultPanel.WithPanels(panels).Srender()
	return panel
}

func (f *DeploymentFormatter) deploymentInfo() string {
	info := pterm.DefaultBox.
		WithLeftPadding(2).
		WithRightPadding(2).
		Sprint(
			pterm.Sprintf("# %s\n\n", i18n.T().Deploying) +
				pterm.Sprintf(i18n.T().DeployVersion+"\n", f.releaseName) +
				pterm.Sprintf(i18n.T().DeployMode+"\n", f.deploymentMode) +
				pterm.Sprintf(i18n.T().DeployStartTime, f.startTime.Format("2006-01-02 15:04:05")),
		)
	return info
}

func (f *DeploymentFormatter) StartStep(step int, description string) {
	if f.beautify {
		f.liveLogs.WriteString(pterm.DefaultSection.Sprintf(i18n.T().DeployStep, fmt.Sprintf("%d/%d", step+1, f.totalSteps), description) + "\n")
		f.updateArea()
	} else {
		fmt.Println(color.Cyan(i18n.T().DeployStep, fmt.Sprintf("%d/%d", step+1, f.totalSteps), description))
	}
}

func (f *DeploymentFormatter) StepLog(step int, log string) {
	if f.beautify {
		f.liveLogs.WriteString(fmt.Sprintf("  - %s\n", log))
		f.updateArea()
	} else {
		fmt.Printf("  - %s\n", log)
	}
}

func (f *DeploymentFormatter) StepSuccess(step int, message string) {
	if f.beautify {
		f.liveLogs.WriteString(pterm.LightGreen(fmt.Sprintf("  ✓ %s\n", message)))
		f.updateArea()
	} else {
		fmt.Println(color.Green(fmt.Sprintf("  ✓ %s", message)))
	}
}

func (f *DeploymentFormatter) StepWarn(step int, message string) {
	if f.beautify {
		f.liveLogs.WriteString(pterm.LightYellow(fmt.Sprintf("  ! %s\n", message)))
		f.updateArea()
	} else {
		fmt.Println(color.Yellow(fmt.Sprintf("  ! %s", message)))
	}
}

func (f *DeploymentFormatter) CompleteDeployment(success bool, finalMessage string) {
	if f.beautify {
		f.ptermArea.Stop()
		f.multiPrinter.Stop()
		if success {
			pterm.Success.Println(i18n.T().DeploySuccess)
		} else {
			pterm.Error.Println(fmt.Sprintf(i18n.T().DeployFailed, finalMessage))
		}
	} else {
		if success {
			fmt.Println(color.Green(i18n.T().DeploySuccess))
		} else {
			fmt.Println(color.Red(fmt.Sprintf(i18n.T().DeployFailed, finalMessage)))
		}
	}
}
