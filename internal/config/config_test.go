package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.App.Name != "myapp" {
		t.Errorf("Expected app name to be 'myapp', got %s", cfg.App.Name)
	}
	
	if cfg.App.KeepReleases != 5 {
		t.Errorf("Expected keep_releases to be 5, got %d", cfg.App.KeepReleases)
	}
	
	if cfg.Server.Port != 22 {
		t.Errorf("Expected port to be 22, got %d", cfg.Server.Port)
	}
	
	if cfg.Deploy.Path != "/opt/myapp" {
		t.Errorf("Expected deploy path to be '/opt/myapp', got %s", cfg.Deploy.Path)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "revlay-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	// Create a test config
	cfg := DefaultConfig()
	cfg.App.Name = "testapp"
	cfg.Server.Host = "test.example.com"
	
	// Save config
	if err := SaveConfig(cfg, tmpFile.Name()); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Load config
	loadedCfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Verify loaded config
	if loadedCfg.App.Name != "testapp" {
		t.Errorf("Expected app name to be 'testapp', got %s", loadedCfg.App.Name)
	}
	
	if loadedCfg.Server.Host != "test.example.com" {
		t.Errorf("Expected server host to be 'test.example.com', got %s", loadedCfg.Server.Host)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	
	// Valid config should pass
	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}
	
	// Invalid app name should fail
	cfg.App.Name = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Empty app name should fail validation")
	}
	
	// Reset and test server host
	cfg = DefaultConfig()
	cfg.Server.Host = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Empty server host should fail validation")
	}
	
	// Reset and test invalid port
	cfg = DefaultConfig()
	cfg.Server.Port = -1
	if err := cfg.Validate(); err == nil {
		t.Error("Invalid port should fail validation")
	}
}

func TestConfigPaths(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Deploy.Path = "/opt/myapp"
	
	if cfg.GetReleasesPath() != "/opt/myapp/releases" {
		t.Errorf("Expected releases path to be '/opt/myapp/releases', got %s", cfg.GetReleasesPath())
	}
	
	if cfg.GetSharedPath() != "/opt/myapp/shared" {
		t.Errorf("Expected shared path to be '/opt/myapp/shared', got %s", cfg.GetSharedPath())
	}
	
	if cfg.GetCurrentPath() != "/opt/myapp/current" {
		t.Errorf("Expected current path to be '/opt/myapp/current', got %s", cfg.GetCurrentPath())
	}
	
	if cfg.GetReleasePathByName("test") != "/opt/myapp/releases/test" {
		t.Errorf("Expected release path to be '/opt/myapp/releases/test', got %s", cfg.GetReleasePathByName("test"))
	}
}