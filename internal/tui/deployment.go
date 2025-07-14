package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xukonxe/revlay/internal/i18n"
)

// DeploymentStatus 表示部署步骤的状态
type StepStatus int

const (
	StatusPending StepStatus = iota
	StatusInProgress
	StatusSuccess
	StatusWarning
	StatusError
)

// DeploymentStep 表示部署过程中的一个步骤
type DeploymentStep struct {
	Name        string
	Description string
	Status      StepStatus
	Message     string
}

// DeploymentModel 是部署过程的 bubbletea 模型
type DeploymentModel struct {
	ReleaseName  string
	DeployMode   string
	Steps        []DeploymentStep
	CurrentStep  int
	Spinner      spinner.Model
	Width        int
	Height       int
	IsDone       bool
	IsSuccess    bool
	ErrorMessage string
	startTime    time.Time
}

// 样式
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginLeft(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			MarginLeft(2)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#39FF14"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5252"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFCA3A"))

	pendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Faint(true)

	stepNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#61AFEF")).
			Bold(true)
)

// NewDeploymentModel 创建一个新的部署模型
func NewDeploymentModel(releaseName, deployMode string) DeploymentModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return DeploymentModel{
		ReleaseName: releaseName,
		DeployMode:  deployMode,
		Steps: []DeploymentStep{
			{Name: i18n.T().DeployPreflightChecks, Status: StatusPending},
			{Name: i18n.T().DeploySetupDirs, Status: StatusPending},
			{Name: i18n.T().DeployStoppingService, Status: StatusPending},
			{Name: i18n.T().DeployActivating, Status: StatusPending},
			{Name: i18n.T().DeployStartingService, Status: StatusPending},
			{Name: i18n.T().DeployHealthCheck, Status: StatusPending},
		},
		CurrentStep: -1,
		Spinner:     s,
		Width:       80,
		Height:      20,
		startTime:   time.Now(),
	}
}

// StartStep 开始执行指定的步骤
func (m *DeploymentModel) StartStep(step int) tea.Cmd {
	if step >= 0 && step < len(m.Steps) {
		m.CurrentStep = step
		m.Steps[step].Status = StatusInProgress
		return m.Spinner.Tick
	}
	return nil
}

// CompleteStep 标记当前步骤为完成
func (m *DeploymentModel) CompleteStep(status StepStatus, message string) {
	if m.CurrentStep >= 0 && m.CurrentStep < len(m.Steps) {
		m.Steps[m.CurrentStep].Status = status
		m.Steps[m.CurrentStep].Message = message
	}
}

// CompleteDeployment 完成整个部署过程
func (m *DeploymentModel) CompleteDeployment(isSuccess bool, errorMessage string) {
	m.IsDone = true
	m.IsSuccess = isSuccess
	m.ErrorMessage = errorMessage
}

// Init 初始化模型
func (m DeploymentModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

// Update 更新模型状态
func (m DeploymentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	// 更新spinner
	var cmd tea.Cmd
	m.Spinner, cmd = m.Spinner.Update(msg)
	return m, cmd
}

// View 渲染模型
func (m DeploymentModel) View() string {
	var s strings.Builder

	// 标题
	s.WriteString(titleStyle.Render(fmt.Sprintf("🚀 部署版本: %s\n", m.ReleaseName)))
	s.WriteString(infoStyle.Render(fmt.Sprintf("模式: %s\n", m.DeployMode)))
	s.WriteString(infoStyle.Render(fmt.Sprintf("已运行: %s\n\n", time.Since(m.startTime).Round(time.Second))))

	// 渲染步骤
	for i, step := range m.Steps {
		var stepDisplay string

		// 渲染步骤号和中文数字
		chineseNumbers := []string{"一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}
		stepNumber := ""
		if i < len(chineseNumbers) {
			stepNumber = chineseNumbers[i]
		} else {
			stepNumber = fmt.Sprintf("%d", i+1)
		}

		stepPrefix := stepNumberStyle.Render(fmt.Sprintf("步骤 %s: ", stepNumber))

		// 根据步骤状态设置样式和图标
		switch step.Status {
		case StatusPending:
			stepDisplay = pendingStyle.Render(fmt.Sprintf("%s %s", stepPrefix, step.Name))
		case StatusInProgress:
			stepDisplay = fmt.Sprintf("%s %s %s", stepPrefix, step.Name, m.Spinner.View())
		case StatusSuccess:
			stepDisplay = successStyle.Render(fmt.Sprintf("%s %s ✓", stepPrefix, step.Name))
			if step.Message != "" {
				stepDisplay += " " + step.Message
			}
		case StatusWarning:
			stepDisplay = warningStyle.Render(fmt.Sprintf("%s %s ⚠️ ", stepPrefix, step.Name))
			if step.Message != "" {
				stepDisplay += " " + warningStyle.Render(step.Message)
			}
		case StatusError:
			stepDisplay = errorStyle.Render(fmt.Sprintf("%s %s ✗", stepPrefix, step.Name))
			if step.Message != "" {
				stepDisplay += " " + errorStyle.Render(step.Message)
			}
		}

		s.WriteString(stepDisplay + "\n")
	}

	// 部署完成信息
	if m.IsDone {
		s.WriteString("\n")
		if m.IsSuccess {
			s.WriteString(successStyle.Render("✅ 部署成功完成!\n"))
		} else {
			s.WriteString(errorStyle.Render(fmt.Sprintf("❌ 部署失败: %s\n", m.ErrorMessage)))
		}
	}

	return s.String()
}

// DeploymentProgressMsg 是用于更新部署进度的消息
type DeploymentProgressMsg struct {
	Step    int
	Status  StepStatus
	Message string
}

// DeploymentDoneMsg 是用于表示部署完成的消息
type DeploymentDoneMsg struct {
	IsSuccess    bool
	ErrorMessage string
}
