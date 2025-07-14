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
	// RootPath is the directory containing the revlay.yml file. It's set at runtime.
	RootPath string `yaml:"-"`

	// Application configuration
	App struct {
		Name         string `yaml:"name"`
		KeepReleases int    `yaml:"keep_releases"`
	} `yaml:"app"`

	// Deployment configuration
	Deploy struct {
		Environment map[string]string `yaml:"environment"`
		Mode        DeploymentMode    `yaml:"mode"`
	} `yaml:"deploy"`

	// Service management configuration
	Service struct {
		// Service start command, ${PORT} will be substituted
		StartCommand string `yaml:"start_command"`
		// Service stop command
		StopCommand string `yaml:"stop_command"`
		// Port the service runs on
		Port int `yaml:"port"`
		// Alternative port for blue-green deployment
		AltPort int `yaml:"alt_port"`
		// Proxy port that listens to public traffic
		ProxyPort int `yaml:"proxy_port"`
		// Health check URL path
		HealthCheck string `yaml:"health_check"`
		// Graceful shutdown timeout in seconds
		GracefulTimeout int `yaml:"graceful_timeout"`
		// Startup confirmation delay in seconds
		StartupDelay int `yaml:"startup_delay"`
		// PID file path
		PidFile string `yaml:"pid_file"`
		// Stdout log path
		StdoutLog string `yaml:"stdout_log"`
		// Stderr log path
		StderrLog string `yaml:"stderr_log"`
	} `yaml:"service"`

	// Hooks configuration
	Hooks struct {
		PreDeploy    []string `yaml:"pre_deploy"`
		PostDeploy   []string `yaml:"post_deploy"`
		PreRollback  []string `yaml:"pre_rollback"`
		PostRollback []string `yaml:"post_rollback"`
	} `yaml:"hooks"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		App: struct {
			Name         string `yaml:"name"`
			KeepReleases int    `yaml:"keep_releases"`
		}{
			Name:         "myapp",
			KeepReleases: 5,
		},
		Deploy: struct {
			Environment map[string]string `yaml:"environment"`
			Mode        DeploymentMode    `yaml:"mode"`
		}{
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
			Mode: ZeroDowntimeMode,
		},
		Service: struct {
			StartCommand    string `yaml:"start_command"`
			StopCommand     string `yaml:"stop_command"`
			Port            int    `yaml:"port"`
			AltPort         int    `yaml:"alt_port"`
			ProxyPort       int    `yaml:"proxy_port"`
			HealthCheck     string `yaml:"health_check"`
			GracefulTimeout int    `yaml:"graceful_timeout"`
			StartupDelay    int    `yaml:"startup_delay"`
			PidFile         string `yaml:"pid_file"`
			StdoutLog       string `yaml:"stdout_log"`
			StderrLog       string `yaml:"stderr_log"`
		}{
			StartCommand:    "systemctl start myapp",
			StopCommand:     "systemctl stop myapp",
			Port:            8080,
			AltPort:         8081,
			ProxyPort:       80,
			HealthCheck:     "/health",
			GracefulTimeout: 30,
			StartupDelay:    10,
			PidFile:         "pids/{{.AppName}}.pid",
			StdoutLog:       "logs/{{.AppName}}-output.log",
			StderrLog:       "logs/{{.AppName}}-error.log",
		},
		Hooks: struct {
			PreDeploy    []string `yaml:"pre_deploy"`
			PostDeploy   []string `yaml:"post_deploy"`
			PreRollback  []string `yaml:"pre_rollback"`
			PostRollback []string `yaml:"post_rollback"`
		}{
			PreDeploy:    []string{},
			PostDeploy:   []string{"systemctl reload nginx"},
			PreRollback:  []string{},
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
		if c.Service.ProxyPort > 0 && (c.Service.ProxyPort == c.Service.Port || c.Service.ProxyPort == c.Service.AltPort) {
			return fmt.Errorf("service.proxy_port must not be the same as port or alt_port")
		}
		if c.Service.Port == c.Service.AltPort {
			return fmt.Errorf("service.port and service.alt_port must be different")
		}
		if c.Service.StartCommand == "" {
			return fmt.Errorf("service.start_command is required for zero_downtime mode")
		}
	}

	return nil
}

// GetStatePath returns the path to the state directory
func (c *Config) GetStatePath() string {
	return filepath.Join(c.RootPath, ".revlay")
}

// GetActivePortPath returns the path to the active port state file
func (c *Config) GetActivePortPath() string {
	return filepath.Join(c.GetStatePath(), "active_port")
}

// GetReleasesPath returns the path to the releases directory
func (c *Config) GetReleasesPath() string {
	return filepath.Join(c.RootPath, "releases")
}

// GetSharedPath returns the path to the shared directory
func (c *Config) GetSharedPath() string {
	return filepath.Join(c.RootPath, "shared")
}

// GetPidsPath returns the path to the pids directory
func (c *Config) GetPidsPath() string {
	return filepath.Join(c.RootPath, "pids")
}

// GetLogsPath returns the path to the logs directory
func (c *Config) GetLogsPath() string {
	return filepath.Join(c.RootPath, "logs")
}

// GetCurrentPath returns the path to the current symlink
func (c *Config) GetCurrentPath() string {
	return filepath.Join(c.RootPath, "current")
}

// GetReleasePathByName returns the path to a specific release
func (c *Config) GetReleasePathByName(release string) string {
	return filepath.Join(c.GetReleasesPath(), release)
}
