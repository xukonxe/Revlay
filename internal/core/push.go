package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/pterm/pterm"
	"github.com/xukonxe/revlay/internal/ssh"
)

// ErrRemoteUpdated is a special error to indicate that the remote was updated and the command should be re-run.
var ErrRemoteUpdated = fmt.Errorf("remote revlay was updated")

// PushOptions defines the options for a push operation.
type PushOptions struct {
	SourceDir    string
	User         string
	Host         string
	Port         int
	KeyFile      string
	AppName      string
	SSHArgs      []string      // Add SSHArgs here
	GetVersion   func() string // Function to get local version
	NewSSHClient func(user, host string, port int, keyFile string, sshArgs []string, verbose bool) ssh.Client
	Quiet        bool
	Verbose      bool
}

// Pusher handles the logic for the push command.
type Pusher struct {
	Opts   *PushOptions
	client ssh.Client
}

// NewPusher creates a new Pusher.
func NewPusher(opts *PushOptions) *Pusher {
	client := opts.NewSSHClient(opts.User, opts.Host, opts.Port, opts.KeyFile, opts.SSHArgs, opts.Verbose)
	return &Pusher{
		Opts:   opts,
		client: client,
	}
}

// Push executes the entire push workflow.
// It returns a channel for status updates and an error channel.
func (p *Pusher) Push() error {
	// Note: The UI/Spinner logic is now handled in the CLI package.
	// This function will focus on the core logic and return errors.

	// 1. Remote probe & version handshake
	remoteVersionStr, err := p.checkRemoteRevlay()
	if err != nil {
		return err
	}
	err = p.handleVersionHandshake(remoteVersionStr)
	if err != nil {
		return err
	}

	// 2. Check remote app
	err = p.checkRemoteAppExists()
	if err != nil {
		return err
	}

	// 3. Create temp dir, rsync, and deploy
	return p.syncAndDeploy()
}

func (p *Pusher) syncAndDeploy() error {
	// Create a temporary directory on the remote server
	remoteTempDir, err := p.client.RunCommand("mktemp -d")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory on remote: %w", err)
	}
	remoteTempDir = strings.TrimSpace(remoteTempDir)

	// Defer cleanup of the temporary directory
	defer p.client.RunCommand(fmt.Sprintf("rm -rf %s", remoteTempDir))

	// Use rsync to push files
	if err := p.client.Rsync(p.Opts.SourceDir, remoteTempDir); err != nil {
		return fmt.Errorf("failed to rsync files: %w", err)
	}

	// Execute remote deploy
	deployCommand := fmt.Sprintf("revlay deploy --from-dir %s --app %s", remoteTempDir, p.Opts.AppName)
	if err := p.client.RunCommandStream(deployCommand); err != nil {
		return fmt.Errorf("remote deployment failed: %w", err)
	}

	return nil
}

func (p *Pusher) checkRemoteRevlay() (string, error) {
	if _, err := p.client.RunCommand("command -v revlay"); err != nil {
		return "", fmt.Errorf("revlay not found on the remote server")
	}
	version, err := p.client.RunCommand("revlay --version")
	if err != nil {
		return "", fmt.Errorf("failed to get remote revlay version: %w", err)
	}
	return strings.TrimSpace(version), nil
}

func (p *Pusher) handleVersionHandshake(remoteVersionStr string) error {
	localVersionStr := p.Opts.GetVersion()
	if localVersionStr == "" {
		if !p.Opts.Quiet {
			pterm.Warning.Println("Local version is not set (development build), skipping compatibility check.")
		}
		return nil
	}

	localVersion, err := semver.ParseTolerant(localVersionStr)
	if err != nil {
		return fmt.Errorf("could not parse local version '%s': %w", localVersionStr, err)
	}
	remoteVersion, err := semver.ParseTolerant(remoteVersionStr)
	if err != nil {
		return fmt.Errorf("could not parse remote version '%s': %w", remoteVersionStr, err)
	}

	if !p.Opts.Quiet {
		pterm.Info.Printf("Version Check: Local %s, Remote %s", localVersion, remoteVersion)
	}

	if localVersion.Major != remoteVersion.Major {
		return fmt.Errorf("major version mismatch (local %d, remote %d)", localVersion.Major, remoteVersion.Major)
	}
	if localVersion.Minor < remoteVersion.Minor {
		return fmt.Errorf("local version is older than remote (local %d.%d, remote %d.%d)", localVersion.Major, localVersion.Minor, remoteVersion.Major, remoteVersion.Minor)
	}
	if localVersion.Minor > remoteVersion.Minor {
		if !p.Opts.Quiet {
			pterm.Warning.Printf("Remote version is older. Attempting automatic update...")
		}
		err := p.client.UpdateRevlay(localVersionStr)
		if err != nil {
			return fmt.Errorf("remote update failed: %w", err)
		}
		if !p.Opts.Quiet {
			pterm.Success.Println("Remote revlay updated. Please run 'revlay push' again.")
		}
		return ErrRemoteUpdated // Signal to the CLI to exit gracefully
	}

	return nil
}

func (p *Pusher) checkRemoteAppExists() error {
	cmd := fmt.Sprintf("revlay service list --output=json")
	output, err := p.client.RunCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to check for remote app '%s': %w", p.Opts.AppName, err)
	}

	var services map[string]interface{}
	if err := json.Unmarshal([]byte(output), &services); err != nil {
		return fmt.Errorf("failed to parse remote service list: %w", err)
	}

	if _, exists := services[p.Opts.AppName]; !exists {
		if p.Opts.Quiet {
			return fmt.Errorf("app '%s' not found and quiet mode is enabled", p.Opts.AppName)
		}
		pterm.Warning.Printf("Application '%s' not found on the remote server.\n", p.Opts.AppName)
		shouldInit, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultValue(false).
			Show("Initialize it now?")
		if !shouldInit {
			return fmt.Errorf("operation cancelled by user")
		}
		pterm.Info.Println("Interactive initialization is not yet implemented.")
		return fmt.Errorf("initialization required")
	}

	return nil
}
