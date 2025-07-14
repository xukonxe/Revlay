package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

// ServiceEntry 表示全局服务列表中的一个服务条目
type ServiceEntry struct {
	Name string `yaml:"name"`
	Root string `yaml:"root"`
}

// ServicesList 表示全局服务列表配置
type ServicesList struct {
	Services map[string]ServiceEntry `yaml:"services"`
}

// DefaultServicesList 返回默认的全局服务列表配置
func DefaultServicesList() *ServicesList {
	return &ServicesList{
		Services: make(map[string]ServiceEntry),
	}
}

// GetServicesConfigPath 返回全局服务列表配置文件的路径
func GetServicesConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取用户主目录，则使用系统临时目录
		return filepath.Join(os.TempDir(), ".revlay", "services.yml")
	}
	return filepath.Join(homeDir, ".revlay", "services.yml")
}

// LoadServicesList 加载全局服务列表配置
func LoadServicesList() (*ServicesList, error) {
	configPath := GetServicesConfigPath()

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// 如果文件不存在，创建一个默认的配置文件
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := DefaultServicesList()
		if err := SaveServicesList(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default services config: %w", err)
		}
		return defaultConfig, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read services config file: %w", err)
	}

	var config ServicesList
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse services config file: %w", err)
	}

	// 确保 Services 字段不为 nil
	if config.Services == nil {
		config.Services = make(map[string]ServiceEntry)
	}

	return &config, nil
}

// SaveServicesList 保存全局服务列表配置
func SaveServicesList(config *ServicesList) error {
	configPath := GetServicesConfigPath()

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 使用文件锁确保并发安全
	fileLock := flock.New(configPath + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock for services config: %w", err)
	}
	if !locked {
		return fmt.Errorf("services config is being modified by another process")
	}
	defer fileLock.Unlock()

	// 序列化并保存配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal services config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write services config file: %w", err)
	}

	return nil
}

// AddService 向全局服务列表中添加一个服务
func AddService(id, name, root string) error {
	config, err := LoadServicesList()
	if err != nil {
		return err
	}

	// 检查服务 ID 是否已存在
	if _, exists := config.Services[id]; exists {
		return fmt.Errorf("service with ID '%s' already exists", id)
	}

	// 检查根目录是否存在
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return fmt.Errorf("service root directory '%s' does not exist", root)
	}

	// 检查根目录中是否存在 revlay.yml 文件
	revlayConfigPath := filepath.Join(root, "revlay.yml")
	if _, err := os.Stat(revlayConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("revlay.yml not found in '%s'", root)
	}

	// 添加服务
	config.Services[id] = ServiceEntry{
		Name: name,
		Root: root,
	}

	// 保存配置
	return SaveServicesList(config)
}

// RemoveService 从全局服务列表中移除一个服务
func RemoveService(id string) error {
	config, err := LoadServicesList()
	if err != nil {
		return err
	}

	// 检查服务是否存在
	if _, exists := config.Services[id]; !exists {
		return fmt.Errorf("service with ID '%s' not found", id)
	}

	// 移除服务
	delete(config.Services, id)

	// 保存配置
	return SaveServicesList(config)
}

// GetService 获取指定 ID 的服务
func GetService(id string) (*ServiceEntry, error) {
	config, err := LoadServicesList()
	if err != nil {
		return nil, err
	}

	// 检查服务是否存在
	service, exists := config.Services[id]
	if !exists {
		return nil, fmt.Errorf("service with ID '%s' not found", id)
	}

	return &service, nil
}

// ListServices 列出所有服务
func ListServices() (map[string]ServiceEntry, error) {
	config, err := LoadServicesList()
	if err != nil {
		return nil, err
	}

	return config.Services, nil
}
