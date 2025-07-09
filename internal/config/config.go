package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	} `yaml:"deploy"`

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
		}{
			Path:        "/opt/myapp",
			SharedPaths: []string{"storage/logs", "storage/uploads"},
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
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