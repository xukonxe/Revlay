package ssh

import (
	"fmt"
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