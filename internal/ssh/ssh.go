package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client represents an SSH client connection
type Client struct {
	conn *ssh.Client
}

// Config represents SSH connection configuration
type Config struct {
	Host     string
	User     string
	Port     int
	Password string
	KeyFile  string
	Timeout  time.Duration
}

// NewClient creates a new SSH client
func NewClient(config *Config) (*Client, error) {
	// Default timeout
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Configure authentication
	var authMethods []ssh.AuthMethod

	// Try key-based authentication first
	if config.KeyFile != "" {
		key, err := loadPrivateKey(config.KeyFile)
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(key))
		}
	}

	// Try password authentication if provided
	if config.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available")
	}

	// Configure SSH client
	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add proper host key verification
		Timeout:         config.Timeout,
	}

	// Connect to SSH server
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	conn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	return &Client{conn: conn}, nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RunCommand executes a command on the remote server
func (c *Client) RunCommand(command string) (string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// UploadFile uploads a file to the remote server
func (c *Client) UploadFile(localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Get file info
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create SCP session
	session, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up SCP command
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		
		// Send file header
		fmt.Fprintf(w, "C%04o %d %s\n", info.Mode()&0777, len(data), filepath.Base(remotePath))
		
		// Send file content
		w.Write(data)
		
		// Send end marker
		fmt.Fprint(w, "\x00")
	}()

	// Execute SCP command
	if err := session.Run("scp -t " + remotePath); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// DownloadFile downloads a file from the remote server
func (c *Client) DownloadFile(remotePath, localPath string) error {
	session, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Execute cat command to get file content
	output, err := session.Output("cat " + remotePath)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Write to local file
	if err := os.WriteFile(localPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

// FileExists checks if a file exists on the remote server
func (c *Client) FileExists(path string) (bool, error) {
	_, err := c.RunCommand("test -f " + path)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DirExists checks if a directory exists on the remote server
func (c *Client) DirExists(path string) (bool, error) {
	_, err := c.RunCommand("test -d " + path)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateDir creates a directory on the remote server
func (c *Client) CreateDir(path string) error {
	_, err := c.RunCommand("mkdir -p " + path)
	return err
}

// RemoveFile removes a file on the remote server
func (c *Client) RemoveFile(path string) error {
	_, err := c.RunCommand("rm -f " + path)
	return err
}

// RemoveDir removes a directory on the remote server
func (c *Client) RemoveDir(path string) error {
	_, err := c.RunCommand("rm -rf " + path)
	return err
}

// CreateSymlink creates a symbolic link on the remote server
func (c *Client) CreateSymlink(target, link string) error {
	_, err := c.RunCommand(fmt.Sprintf("ln -sfn %s %s", target, link))
	return err
}

// ReadSymlink reads a symbolic link on the remote server
func (c *Client) ReadSymlink(path string) (string, error) {
	output, err := c.RunCommand("readlink " + path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// ListFiles lists files in a directory on the remote server
func (c *Client) ListFiles(path string) ([]string, error) {
	output, err := c.RunCommand("ls -1 " + path)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	
	return files, nil
}

// loadPrivateKey loads a private key from file
func loadPrivateKey(path string) (ssh.Signer, error) {
	// Expand tilde
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, path[2:])
	}

	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// TestConnection tests the SSH connection
func (c *Client) TestConnection() error {
	_, err := c.RunCommand("echo 'Connection test successful'")
	return err
}