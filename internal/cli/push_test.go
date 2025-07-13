package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xukonxe/revlay/internal/ssh"
)

// mockSSHClient is a mock implementation of the ssh.Client interface for testing.
type mockSSHClient struct {
	// A map to store expected commands and their mock outputs.
	mockRunCommand func(command string) (string, error)
	// Mock implementation for RunCommandStream.
	mockRunCommandStream func(command string) error
	// Mock implementation for Rsync.
	mockRsync func(sourceDir, remoteDir string) error
}

func (m *mockSSHClient) RunCommand(command string) (string, error) {
	if m.mockRunCommand != nil {
		return m.mockRunCommand(command)
	}
	return "", nil
}

func (m *mockSSHClient) RunCommandStream(command string) error {
	if m.mockRunCommandStream != nil {
		return m.mockRunCommandStream(command)
	}
	return nil
}

func (m *mockSSHClient) Rsync(sourceDir, remoteDir string) error {
	if m.mockRsync != nil {
		return m.mockRsync(sourceDir, remoteDir)
	}
	return nil
}

// Override the newSSHClient factory for testing.
func setupPushTest(mockClient ssh.Client) {
	newSSHClient = func(user, host string) ssh.Client {
		return mockClient
	}
}

func TestRunPush_Success(t *testing.T) {
	mock := &mockSSHClient{
		mockRunCommand: func(cmd string) (string, error) {
			if strings.Contains(cmd, "mktemp -d") {
				return "/tmp/tempdir123", nil
			}
			if strings.Contains(cmd, "command -v revlay") {
				return "/usr/local/bin/revlay", nil
			}
			if strings.Contains(cmd, "rm -rf") {
				return "", nil
			}
			return "", fmt.Errorf("unexpected command: %s", cmd)
		},
		mockRunCommandStream: func(cmd string) error {
			assert.Equal(t, "revlay deploy --from-dir /tmp/tempdir123 my-app", cmd)
			return nil
		},
		mockRsync: func(source, dest string) error {
			assert.Equal(t, "./dist", source)
			assert.Equal(t, "/tmp/tempdir123", dest)
			return nil
		},
	}
	setupPushTest(mock)

	cmd := NewPushCommand()
	cmd.SetArgs([]string{"./dist", "to", "user@host", "--to", "my-app"})
	err := cmd.Execute()

	assert.NoError(t, err)
}

func TestRunPush_RevlayNotFound(t *testing.T) {
	mock := &mockSSHClient{
		mockRunCommand: func(cmd string) (string, error) {
			if strings.Contains(cmd, "command -v revlay") {
				return "", fmt.Errorf("command not found")
			}
			return "", nil
		},
	}
	setupPushTest(mock)

	cmd := NewPushCommand()
	cmd.SetArgs([]string{"./dist", "to", "user@host", "--to", "my-app"})
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revlay not found on the remote server")
}

func TestRunPush_MktempFails(t *testing.T) {
	mock := &mockSSHClient{
		mockRunCommand: func(cmd string) (string, error) {
			if strings.Contains(cmd, "command -v revlay") {
				return "/usr/local/bin/revlay", nil
			}
			if strings.Contains(cmd, "mktemp -d") {
				return "", fmt.Errorf("mktemp failed")
			}
			return "", nil
		},
	}
	setupPushTest(mock)

	cmd := NewPushCommand()
	cmd.SetArgs([]string{"./dist", "to", "user@host", "--to", "my-app"})
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temporary directory")
}

func TestRunPush_RsyncFails(t *testing.T) {
	mock := &mockSSHClient{
		mockRunCommand: func(cmd string) (string, error) {
			if strings.Contains(cmd, "command -v revlay") {
				return "/usr/local/bin/revlay", nil
			}
			if strings.Contains(cmd, "mktemp -d") {
				return "/tmp/tempdir123", nil
			}
			return "", nil
		},
		mockRsync: func(source, dest string) error {
			return fmt.Errorf("rsync error")
		},
	}
	setupPushTest(mock)

	cmd := NewPushCommand()
	cmd.SetArgs([]string{"./dist", "to", "user@host", "--to", "my-app"})
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rsync files")
}

func TestParseDestination(t *testing.T) {
	testCases := []struct {
		name         string
		dest         string
		expectedUser string
		expectedHost string
		expectErr    bool
	}{
		{"user and host", "user@host.com", "user", "host.com", false},
		{"host only", "host.com", "", "host.com", false},
		{"invalid format", "user@", "", "", true},
		{"another invalid", "@host.com", "", "", true},
		{"empty", "", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user, host, err := parseDestination(tc.dest)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedUser, user)
				assert.Equal(t, tc.expectedHost, host)
			}
		})
	}
}
