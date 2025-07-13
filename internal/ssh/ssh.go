package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xukonxe/revlay/internal/color"
)

// Client represents an SSH client for a specific host.
type Client struct {
	User string
	Host string
}

// NewClient creates a new SSH client.
func NewClient(user, host string) *Client {
	return &Client{User: user, Host: host}
}

// buildArgs constructs the arguments for the ssh command.
func (c *Client) buildArgs(remoteCommand string) []string {
	dest := c.Host
	if c.User != "" {
		dest = fmt.Sprintf("%s@%s", c.User, c.Host)
	}
	return []string{dest, remoteCommand}
}

// RunCommand executes a command on the remote host via SSH.
func (c *Client) RunCommand(command string) (string, error) {
	args := c.buildArgs(command)
	fmt.Println(color.Cyan("  -> Running on remote: ssh %s", strings.Join(args, " ")))

	cmd := exec.Command("ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh command failed: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

// RunCommandStream executes a command on the remote host via SSH and streams the output.
func (c *Client) RunCommandStream(command string) error {
	args := c.buildArgs(command)
	fmt.Println(color.Cyan("  -> Running on remote: ssh %s", strings.Join(args, " ")))

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh stream command failed: %w", err)
	}
	return nil
}

// Rsync copies a local directory to a remote host.
func (c *Client) Rsync(sourceDir, remoteDir string) error {
	dest := fmt.Sprintf("%s:%s", c.Host, remoteDir)
	if c.User != "" {
		dest = fmt.Sprintf("%s@%s:%s", c.User, c.Host, remoteDir)
	}

	// Ensure sourceDir has a trailing slash to copy contents, not the directory itself
	if !strings.HasSuffix(sourceDir, "/") {
		sourceDir += "/"
	}

	args := []string{
		"-r",         // Recursive
		"-a",         // Archive mode (preserves permissions, etc.)
		"--verbose",  // Show what's happening
		"--delete",   // Delete files on the destination that don't exist on the source
		sourceDir,
		dest,
	}

	fmt.Println(color.Cyan("  -> Running: rsync %s", strings.Join(args, " ")))
	cmd := exec.Command("rsync", args...)
	cmd.Stdout = os.Stdout // Pipe rsync output directly to our stdout
	cmd.Stderr = os.Stderr // And stderr too

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync command failed: %w", err)
	}
	return nil
}