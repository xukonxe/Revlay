package tui

import (
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// Manager 管理 TUI 界面
type Manager struct {
	model       tea.Model
	program     *tea.Program
	initialized bool
	mutex       sync.Mutex
}

var (
	// 单例实例
	instance *Manager
	once     sync.Once
)

// GetManager 返回 Manager 的单例实例
func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			initialized: false,
		}
	})
	return instance
}

// InitDeployment 初始化部署 UI
func (m *Manager) InitDeployment(releaseName, deployMode string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	model := NewDeploymentModel(releaseName, deployMode)
	m.model = model

	// 创建程序
	m.program = tea.NewProgram(model)
	m.initialized = true

	// 在新的 goroutine 中运行程序
	go func() {
		if _, err := m.program.Run(); err != nil {
			os.Stderr.WriteString("Error running program: " + err.Error() + "\n")
			os.Exit(1)
		}
	}()
}

// StartStep 开始执行步骤
func (m *Manager) StartStep(step int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return
	}

	if model, ok := m.model.(DeploymentModel); ok {
		cmd := model.StartStep(step)
		m.program.Send(cmd())
	}
}

// CompleteStep 完成当前步骤
func (m *Manager) CompleteStep(status StepStatus, message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return
	}

	if model, ok := m.model.(*DeploymentModel); ok {
		model.CompleteStep(status, message)
	}
}

// CompleteDeployment 完成部署
func (m *Manager) CompleteDeployment(isSuccess bool, errorMessage string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return
	}

	if model, ok := m.model.(*DeploymentModel); ok {
		model.CompleteDeployment(isSuccess, errorMessage)
	}
}

// Quit 退出程序
func (m *Manager) Quit() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return
	}

	m.program.Quit()
	m.initialized = false
}
