package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DeploymentMode represents different deployment strategies
type DeploymentMode string

const (
	// ZeroDowntimeMode uses blue-green deployment with port switching
	ZeroDowntimeMode DeploymentMode = "zero_downtime"
	// ShortDowntimeMode uses traditional deployment with service restart
	ShortDowntimeMode DeploymentMode = "short_downtime"
)

// Config represents the main configuration structure for revlay.yml
type Config struct {
	// Application configuration
	App struct {
		Name        string `yaml:"name"`
		Repository  string `yaml:"repository"`
		Branch      string `yaml:"branch"`
		KeepReleases int    `yaml:"keep_releases"`
	} `yaml:"app"`

	// Server configuration
	Server struct {
		Host     string `yaml:"host"`
		User     string `yaml:"user"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password,omitempty"`
		KeyFile  string `yaml:"key_file,omitempty"`
	} `yaml:"server"`

	// Deployment configuration
	Deploy struct {
		Path        string            `yaml:"path"`
		SharedPaths []string          `yaml:"shared_paths"`
		Environment map[string]string `yaml:"environment"`
		Mode        DeploymentMode    `yaml:"mode"`
	} `yaml:"deploy"`

	// Service management configuration
	Service struct {
		// Service command to manage (e.g., "systemctl restart myapp")
		Command string `yaml:"command"`
		// Port the service runs on
		Port int `yaml:"port"`
		// Alternative port for blue-green deployment
		AltPort int `yaml:"alt_port"`
		// Health check URL path
		HealthCheck string `yaml:"health_check"`
		// Restart delay in seconds
		RestartDelay int `yaml:"restart_delay"`
		// Graceful shutdown timeout in seconds
		GracefulTimeout int `yaml:"graceful_timeout"`
	} `yaml:"service"`

	// Hooks configuration
	Hooks struct {
		PreDeploy   []string `yaml:"pre_deploy"`
		PostDeploy  []string `yaml:"post_deploy"`
		PreRollback []string `yaml:"pre_rollback"`
		PostRollback []string `yaml:"post_rollback"`
	} `yaml:"hooks"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		App: struct {
			Name        string `yaml:"name"`
			Repository  string `yaml:"repository"`
			Branch      string `yaml:"branch"`
			KeepReleases int    `yaml:"keep_releases"`
		}{
			Name:        "myapp",
			Repository:  "",
			Branch:      "main",
			KeepReleases: 5,
		},
		Server: struct {
			Host     string `yaml:"host"`
			User     string `yaml:"user"`
			Port     int    `yaml:"port"`
			Password string `yaml:"password,omitempty"`
			KeyFile  string `yaml:"key_file,omitempty"`
		}{
			Host:    "localhost",
			User:    "deploy",
			Port:    22,
			KeyFile: "~/.ssh/id_rsa",
		},
		Deploy: struct {
			Path        string            `yaml:"path"`
			SharedPaths []string          `yaml:"shared_paths"`
			Environment map[string]string `yaml:"environment"`
			Mode        DeploymentMode    `yaml:"mode"`
		}{
			Path:        "/opt/myapp",
			SharedPaths: []string{"storage/logs", "storage/uploads"},
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
			Mode: ZeroDowntimeMode,
		},
		Service: struct {
			Command string `yaml:"command"`
			Port int `yaml:"port"`
			AltPort int `yaml:"alt_port"`
			HealthCheck string `yaml:"health_check"`
			RestartDelay int `yaml:"restart_delay"`
			GracefulTimeout int `yaml:"graceful_timeout"`
		}{
			Command: "systemctl restart myapp",
			Port: 8080,
			AltPort: 8081,
			HealthCheck: "/health",
			RestartDelay: 5,
			GracefulTimeout: 30,
		},
		Hooks: struct {
			PreDeploy   []string `yaml:"pre_deploy"`
			PostDeploy  []string `yaml:"post_deploy"`
			PreRollback []string `yaml:"pre_rollback"`
			PostRollback []string `yaml:"post_rollback"`
		}{
			PreDeploy:   []string{},
			PostDeploy:  []string{"systemctl reload nginx"},
			PreRollback: []string{},
			PostRollback: []string{"systemctl reload nginx"},
		},
	}
}

// LoadConfig loads configuration from revlay.yml file
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "revlay.yml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to revlay.yml file
func SaveConfig(config *Config, path string) error {
	if path == "" {
		path = "revlay.yml"
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server.host is required")
	}
	if c.Server.User == "" {
		return fmt.Errorf("server.user is required")
	}
	if c.Deploy.Path == "" {
		return fmt.Errorf("deploy.path is required")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.App.KeepReleases < 1 {
		return fmt.Errorf("app.keep_releases must be at least 1")
	}
	
	// Validate deployment mode
	if c.Deploy.Mode != "" && c.Deploy.Mode != ZeroDowntimeMode && c.Deploy.Mode != ShortDowntimeMode {
		return fmt.Errorf("deploy.mode must be 'zero_downtime' or 'short_downtime'")
	}
	
	// Set default deployment mode if not specified
	if c.Deploy.Mode == "" {
		c.Deploy.Mode = ZeroDowntimeMode
	}
	
	// Validate service configuration for zero downtime mode
	if c.Deploy.Mode == ZeroDowntimeMode {
		if c.Service.Port <= 0 || c.Service.Port > 65535 {
			return fmt.Errorf("service.port must be between 1 and 65535 for zero downtime deployment")
		}
		if c.Service.AltPort <= 0 || c.Service.AltPort > 65535 {
			return fmt.Errorf("service.alt_port must be between 1 and 65535 for zero downtime deployment")
		}
		if c.Service.Port == c.Service.AltPort {
			return fmt.Errorf("service.port and service.alt_port must be different")
		}
	}
	
	return nil
}

// GetReleasesPath returns the path to the releases directory
func (c *Config) GetReleasesPath() string {
	return filepath.Join(c.Deploy.Path, "releases")
}

// GetSharedPath returns the path to the shared directory
func (c *Config) GetSharedPath() string {
	return filepath.Join(c.Deploy.Path, "shared")
}

// GetCurrentPath returns the path to the current symlink
func (c *Config) GetCurrentPath() string {
	return filepath.Join(c.Deploy.Path, "current")
}

// GetReleasePathByName returns the path to a specific release
func (c *Config) GetReleasePathByName(release string) string {
	return filepath.Join(c.GetReleasesPath(), release)
}