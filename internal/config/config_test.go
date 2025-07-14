package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "myapp", cfg.App.Name)
	assert.Equal(t, 5, cfg.App.KeepReleases)
	assert.Equal(t, ZeroDowntimeMode, cfg.Deploy.Mode)
	assert.Equal(t, 8080, cfg.Service.Port)
	assert.Equal(t, 8081, cfg.Service.AltPort)
	assert.Equal(t, 80, cfg.Service.ProxyPort)
	assert.Equal(t, "systemctl start myapp", cfg.Service.StartCommand)
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "revlay-test-*.yml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := DefaultConfig()
	cfg.App.Name = "testapp"
	cfg.Service.Port = 9000
	cfg.Service.ProxyPort = 88

	err = SaveConfig(cfg, tmpFile.Name())
	assert.NoError(t, err)

	loadedCfg, err := LoadConfig(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "testapp", loadedCfg.App.Name)
	assert.Equal(t, 9000, loadedCfg.Service.Port)
	assert.Equal(t, 88, loadedCfg.Service.ProxyPort)
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid config should pass", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid app name should fail", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.App.Name = ""
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid deploy mode should fail", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Deploy.Mode = "invalid_mode"
		assert.Error(t, cfg.Validate())
	})

	t.Run("zero_downtime mode validation", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Deploy.Mode = ZeroDowntimeMode

		// Invalid port
		cfg.Service.Port = -1
		assert.Error(t, cfg.Validate())
		cfg.Service.Port = 8080

		// Invalid alt_port
		cfg.Service.AltPort = 0
		assert.Error(t, cfg.Validate())
		cfg.Service.AltPort = 8081

		// Same port and alt_port
		cfg.Service.AltPort = 8080
		assert.Error(t, cfg.Validate())
		cfg.Service.AltPort = 8081

		// Proxy port same as service port
		cfg.Service.ProxyPort = 8080
		assert.Error(t, cfg.Validate())
		cfg.Service.ProxyPort = 80

		// Missing start command
		cfg.Service.StartCommand = ""
		assert.Error(t, cfg.Validate())
	})
}

func TestPathGetters(t *testing.T) {
	cfg := &Config{RootPath: "/tmp/my-app"}
	assert.Equal(t, "/tmp/my-app/releases", cfg.GetReleasesPath())
	assert.Equal(t, "/tmp/my-app/shared", cfg.GetSharedPath())
	assert.Equal(t, "/tmp/my-app/current", cfg.GetCurrentPath())
	assert.Equal(t, "/tmp/my-app/.revlay", cfg.GetStatePath())
	assert.Equal(t, "/tmp/my-app/.revlay/active_port", cfg.GetActivePortPath())
	assert.Equal(t, "/tmp/my-app/releases/v1", cfg.GetReleasePathByName("v1"))
}
