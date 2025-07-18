package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
)

// Client defines the interface for an SSH client.
type Client interface {
	RunCommand(command string) (string, error)
	RunCommandStream(command string) error
	Rsync(sourceDir, remoteDir string) error
	UpdateRevlay(localVersion string) error
}

// sshClient implements the Client interface for a specific host.
type sshClient struct {
	User    string
	Host    string
	Port    int
	KeyFile string
	Verbose bool
}

// NewClient creates a new SSH client.
func NewClient(user, host string, port int, keyFile string, verbose bool) Client {
	return &sshClient{User: user, Host: host, Port: port, KeyFile: keyFile, Verbose: verbose}
}

// buildSSHArgs constructs the common arguments for ssh/rsync commands.
func (c *sshClient) buildSSHArgs() []string {
	var args []string
	if c.Port != 0 && c.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", c.Port))
	}
	if c.KeyFile != "" {
		args = append(args, "-i", c.KeyFile)
	}
	return args
}

// buildArgs constructs the arguments for the ssh command.
func (c *sshClient) buildArgs(remoteCommand string) []string {
	dest := c.Host
	if c.User != "" {
		dest = fmt.Sprintf("%s@%s", c.User, c.Host)
	}
	args := c.buildSSHArgs()
	args = append(args, dest, remoteCommand)
	return args
}

// RunCommand executes a command on the remote host via SSH.
func (c *sshClient) RunCommand(command string) (string, error) {
	args := c.buildArgs(command)
	if c.Verbose {
		fmt.Println(color.Cyan(i18n.T().SSHRunningRemote, "ssh "+strings.Join(args, " ")))
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(i18n.T().SSHCommandFailed, err, string(output))
	}
	return string(output), nil
}

// RunCommandStream executes a command on the remote host via SSH and streams the output.
func (c *sshClient) RunCommandStream(command string) error {
	args := c.buildArgs(command)
	if c.Verbose {
		fmt.Println(color.Cyan(i18n.T().SSHRunningRemote, "ssh "+strings.Join(args, " ")))
	}

	cmd := exec.Command("ssh", args...)

	if c.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(i18n.T().SSHStreamFailed, err)
	}
	return nil
}

// Rsync copies a local directory to a remote host.
func (c *sshClient) Rsync(sourceDir, remoteDir string) error {
	dest := fmt.Sprintf("%s:%s", c.Host, remoteDir)
	if c.User != "" {
		dest = fmt.Sprintf("%s@%s:%s", c.User, c.Host, remoteDir)
	}

	if !strings.HasSuffix(sourceDir, "/") {
		sourceDir += "/"
	}

	rsyncArgs := []string{
		"-r", // Recursive
		"-a", // Archive mode
	}

	// Set verbosity or quietness
	if c.Verbose {
		rsyncArgs = append(rsyncArgs, "-v")
	} else {
		rsyncArgs = append(rsyncArgs, "-q")
	}

	// Add custom ssh options if needed
	sshArgs := c.buildSSHArgs()
	if len(sshArgs) > 0 {
		rsyncArgs = append(rsyncArgs, "-e", fmt.Sprintf("ssh %s", strings.Join(sshArgs, " ")))
	}

	rsyncArgs = append(rsyncArgs, sourceDir, dest)

	if c.Verbose {
		fmt.Println(color.Cyan(i18n.T().SSHRsyncCommand, "rsync "+strings.Join(rsyncArgs, " ")))
	}
	cmd := exec.Command("rsync", rsyncArgs...)

	if c.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(i18n.T().SSHRsyncFailed, err)
	}
	return nil
}

// UpdateRevlay handles the process of updating revlay on the remote server.
func (c *sshClient) UpdateRevlay(localVersion string) error {
	pterm.Info.Println("Starting remote revlay update process...")

	// 1. Get remote system info
	pterm.Info.Println("Detecting remote system architecture...")
	archCmd := "uname -s && uname -m"
	archOutput, err := c.RunCommand(archCmd)
	if err != nil {
		return fmt.Errorf("failed to get remote system info: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(archOutput), "\n")
	if len(parts) < 2 {
		return fmt.Errorf("unexpected output from uname: %s", archOutput)
	}
	osType := strings.ToLower(parts[0])
	arch := strings.ToLower(parts[1])
	if arch == "x86_64" {
		arch = "amd64"
	}
	pterm.Success.Printf("Remote system detected: %s/%s\n", osType, arch)

	// 2. Construct download URL
	releaseName := fmt.Sprintf("revlay_%s_%s_%s", strings.TrimPrefix(localVersion, "v"), osType, arch)
	downloadURL := fmt.Sprintf("https://github.com/xukonxe/Revlay/releases/download/%s/%s.tar.gz", localVersion, releaseName)
	pterm.Info.Printf("Constructed download URL: %s\n", downloadURL)

	// 3. Download to a temporary directory
	pterm.Info.Println("Downloading new version to temporary directory...")
	remoteTempFile := "/tmp/revlay_new.tar.gz"

	downloader := ""
	if _, err := c.RunCommand("command -v curl"); err == nil {
		downloader = "curl"
	} else if _, err := c.RunCommand("command -v wget"); err == nil {
		downloader = "wget"
	} else {
		return fmt.Errorf("neither 'curl' nor 'wget' found on the remote server")
	}

	var downloadCmd string
	if downloader == "curl" {
		downloadCmd = fmt.Sprintf("curl -L -o %s %s", remoteTempFile, downloadURL)
	} else {
		downloadCmd = fmt.Sprintf("wget -O %s %s", remoteTempFile, downloadURL)
	}

	if err := c.RunCommandStream(downloadCmd); err != nil {
		return fmt.Errorf("failed to download new version from %s: %w", downloadURL, err)
	}
	pterm.Success.Println("Download complete.")

	// --- Start Atomic Update ---
	pterm.Info.Println("Starting atomic update...")

	// 4. Find existing revlay path and prepare for backup
	revlayPath, err := c.RunCommand("command -v revlay")
	if err != nil {
		return fmt.Errorf("could not find existing revlay path on remote: %w", err)
	}
	revlayPath = strings.TrimSpace(revlayPath)
	backupPath := revlayPath + ".bak"

	// Cleanup function to be called on success or failure
	cleanup := func() {
		pterm.Info.Println("Cleaning up temporary files...")
		c.RunCommand(fmt.Sprintf("rm -f %s", remoteTempFile))
		c.RunCommand(fmt.Sprintf("rm -rf %s", backupPath))
		c.RunCommand(fmt.Sprintf("rm -rf /tmp/revlay_new_unpacked"))
	}

	// Rollback function in case of failure
	rollback := func() {
		pterm.Warning.Println("An error occurred. Rolling back...")
		if _, err := c.RunCommand(fmt.Sprintf("test -f %s", backupPath)); err == nil {
			if _, err := c.RunCommand(fmt.Sprintf("mv %s %s", backupPath, revlayPath)); err != nil {
				pterm.Error.Printf("CRITICAL: Rollback failed! Remote revlay might be in a broken state at '%s'. Please restore it manually from '%s'.\n", revlayPath, backupPath)
			} else {
				pterm.Success.Println("Rollback successful.")
			}
		}
		cleanup()
	}

	// 5. Backup existing binary
	pterm.Info.Printf("Backing up existing binary from '%s' to '%s'...\n", revlayPath, backupPath)
	if _, err := c.RunCommand(fmt.Sprintf("mv %s %s", revlayPath, backupPath)); err != nil {
		pterm.Error.Printf("Failed to backup existing revlay: %v\n", err)
		cleanup() // Clean up downloaded file
		return err
	}

	// 6. Unpack and verify
	pterm.Info.Println("Unpacking new version...")
	unpackDir := "/tmp/revlay_new_unpacked"
	if _, err := c.RunCommand(fmt.Sprintf("mkdir -p %s && tar -xzf %s -C %s", unpackDir, remoteTempFile, unpackDir)); err != nil {
		rollback()
		return fmt.Errorf("failed to unpack new version: %w", err)
	}

	newBinaryPath := fmt.Sprintf("%s/revlay", unpackDir)
	pterm.Info.Println("Verifying new binary...")
	// Make executable
	if _, err := c.RunCommand(fmt.Sprintf("chmod +x %s", newBinaryPath)); err != nil {
		rollback()
		return fmt.Errorf("failed to make new binary executable: %w", err)
	}
	// Check version
	output, err := c.RunCommand(fmt.Sprintf("%s --version", newBinaryPath))
	if err != nil {
		rollback()
		return fmt.Errorf("failed to verify new binary version: %w", err)
	}
	newVersionStr := strings.TrimSpace(output)
	pterm.Success.Printf("New binary verified. Version: %s\n", newVersionStr)

	// 7. Atomic replace
	pterm.Info.Printf("Replacing old binary with new one at '%s'...\n", revlayPath)
	if _, err := c.RunCommand(fmt.Sprintf("mv %s %s", newBinaryPath, revlayPath)); err != nil {
		rollback()
		return fmt.Errorf("failed to move new binary into place: %w", err)
	}

	// 8. Success and cleanup
	pterm.Success.Println("Remote revlay updated successfully!")
	cleanup()

	return nil
}
