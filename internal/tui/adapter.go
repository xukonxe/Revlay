package tui

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// OutputCapture 用于捕获输出并更新 TUI
type OutputCapture struct {
	manager *Manager
	buffer  []byte
	mutex   sync.Mutex
}

var (
	// 单例
	stdoutCapture *OutputCapture
	stderrCapture *OutputCapture
	captureOnce   sync.Once

	// 原始和多路输出
	originalStdout io.Writer
	originalStderr io.Writer
	multiStdout    io.Writer
	multiStderr    io.Writer
)

// SetupOutputCapture 设置输出捕获
func SetupOutputCapture() {
	captureOnce.Do(func() {
		// 保存原始输出
		originalStdout = os.Stdout
		originalStderr = os.Stderr

		// 创建捕获器
		stdoutCapture = &OutputCapture{
			manager: GetManager(),
		}

		stderrCapture = &OutputCapture{
			manager: GetManager(),
		}

		// 创建多路输出
		multiStdout = io.MultiWriter(originalStdout, stdoutCapture)
		multiStderr = io.MultiWriter(originalStderr, stderrCapture)
	})
}

// RestoreOutputs 恢复原始输出
func RestoreOutputs() {
	// 没有更多操作，因为我们不能直接替换 os.Stdout
}

// Write 实现 io.Writer 接口
func (c *OutputCapture) Write(p []byte) (n int, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 缓存输出用于后续处理
	c.buffer = append(c.buffer, p...)

	// 这里可以处理输出并更新 TUI
	// 例如解析输出内容，找出错误、警告等

	return len(p), nil
}

// GetOriginalStdout 获取原始的标准输出
func GetOriginalStdout() io.Writer {
	if originalStdout != nil {
		return originalStdout
	}
	return os.Stdout
}

// GetOriginalStderr 获取原始的标准错误输出
func GetOriginalStderr() io.Writer {
	if originalStderr != nil {
		return originalStderr
	}
	return os.Stderr
}

// PrintToOriginal 直接打印到原始输出，绕过 TUI
func PrintToOriginal(format string, args ...interface{}) {
	fmt.Fprintf(GetOriginalStdout(), format, args...)
}

// GetMultiStdout 获取多路标准输出
func GetMultiStdout() io.Writer {
	if multiStdout != nil {
		return multiStdout
	}
	return os.Stdout
}

// GetMultiStderr 获取多路标准错误输出
func GetMultiStderr() io.Writer {
	if multiStderr != nil {
		return multiStderr
	}
	return os.Stderr
}

// PrintCaptured 打印到多路输出
func PrintCaptured(format string, args ...interface{}) {
	fmt.Fprintf(GetMultiStdout(), format, args...)
}
