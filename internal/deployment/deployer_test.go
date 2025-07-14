package deployment

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xukonxe/revlay/internal/config"
)

// setupTestEnv creates a temporary directory structure and a valid config for testing.
func setupTestEnv(t *testing.T, mode config.DeploymentMode) (*config.Config, string) {
	tmpDir, err := os.MkdirTemp("", "revlay-deploy-test")
	require.NoError(t, err)

	cfg := config.DefaultConfig()
	cfg.RootPath = tmpDir
	cfg.Deploy.Mode = mode
	// Use simple file-based commands for testing side effects that are os-agnostic.
	// We manually substitute ${PORT} in the test assertions.
	cfg.Service.StartCommand = "touch " + filepath.Join(tmpDir, "service_started_on_${PORT}")
	cfg.Service.StopCommand = "touch " + filepath.Join(tmpDir, "service_stopped_on_${PORT}")
	cfg.Service.HealthCheck = "/health"
	cfg.Service.GracefulTimeout = 0 // Disable for tests to avoid waiting
	cfg.App.KeepReleases = 1

	// Disable hooks for testing to avoid os-specific commands like systemctl
	cfg.Hooks.PreDeploy = []string{}
	cfg.Hooks.PostDeploy = []string{}
	cfg.Hooks.PreRollback = []string{}
	cfg.Hooks.PostRollback = []string{}

	return cfg, tmpDir
}

func TestDeployShortDowntime(t *testing.T) {
	cfg, tmpDir := setupTestEnv(t, config.ShortDowntimeMode)
	defer os.RemoveAll(tmpDir)

	deployer := NewLocalDeployer(cfg)
	releaseName := "release-short-1"

	// Mock health check endpoint
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	port, _ := strconv.Atoi(strings.Split(backend.URL, ":")[2])
	cfg.Service.Port = port

	// --- Execute Deployment ---
	err := deployer.Deploy(releaseName, "")
	require.NoError(t, err)

	// --- Assertions ---
	// 1. Check if stop command was "executed" with the correct port
	assert.FileExists(t, filepath.Join(tmpDir, fmt.Sprintf("service_stopped_on_%d", cfg.Service.Port)))

	// 2. Check if symlink is correct
	currentLink, err := os.Readlink(cfg.GetCurrentPath())
	require.NoError(t, err)
	assert.Equal(t, cfg.GetReleasePathByName(releaseName), currentLink)

	// 3. Check if start command was "executed" with the correct port
	assert.FileExists(t, filepath.Join(tmpDir, fmt.Sprintf("service_started_on_%d", cfg.Service.Port)))
}

func TestDeployZeroDowntime(t *testing.T) {
	cfg, tmpDir := setupTestEnv(t, config.ZeroDowntimeMode)
	defer os.RemoveAll(tmpDir)

	// --- Setup mock services and initial state ---
	newService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer newService.Close()
	newPort, _ := strconv.Atoi(strings.Split(newService.URL, ":")[2])

	cfg.Service.AltPort = newPort

	oldPort := cfg.Service.Port
	err := os.MkdirAll(cfg.GetStatePath(), 0755)
	require.NoError(t, err)
	err = os.WriteFile(cfg.GetActivePortPath(), []byte(strconv.Itoa(oldPort)), 0644)
	require.NoError(t, err)

	dummyOldRelease := "release-old"
	err = os.MkdirAll(cfg.GetReleasePathByName(dummyOldRelease), 0755)
	require.NoError(t, err)

	deployer := NewLocalDeployer(cfg)
	releaseName := "release-zero-1"

	// --- Execute Deployment ---
	err = deployer.Deploy(releaseName, "")
	require.NoError(t, err)

	// --- Assertions ---
	// 1. Check if new service was started on alt_port
	assert.FileExists(t, filepath.Join(tmpDir, fmt.Sprintf("service_started_on_%d", newPort)))

	// 2. Check if state file was updated to new port
	stateBytes, err := os.ReadFile(cfg.GetActivePortPath())
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(newPort), string(stateBytes))

	// 3. Check symlink
	currentLink, err := os.Readlink(cfg.GetCurrentPath())
	require.NoError(t, err)
	assert.Equal(t, cfg.GetReleasePathByName(releaseName), currentLink)

	// 4. Check if old service was stopped on main port
	assert.FileExists(t, filepath.Join(tmpDir, fmt.Sprintf("service_stopped_on_%d", oldPort)))

	// 5. Check if old release was pruned
	_, err = os.Stat(cfg.GetReleasePathByName(dummyOldRelease))
	assert.True(t, os.IsNotExist(err), "Old release should have been pruned")
}
