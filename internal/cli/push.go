package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/core"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ssh"
)

var (
	// newSSHClient is a factory function for creating an ssh.Client.
	// It's a variable so it can be replaced in tests.
	newSSHClient = func(user, host string, port int, keyFile string, verbose bool) ssh.Client {
		return ssh.NewClient(user, host, port, keyFile, verbose)
	}
)

// NewPushCommand creates the `revlay push` command.
func NewPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: i18n.T().PushShortDesc,
		Long:  i18n.T().PushLongDesc,
		Example: `  # Push the current directory to the 'production' app on the remote server
  revlay push -p . -to deploy@192.168.1.100 -app production`,
		RunE: runPush,
	}

	cmd.Flags().StringP("path", "p", "", "Path to the local directory to push")
	cmd.Flags().String("to", "", "The destination server in [user@]host format")
	cmd.Flags().String("app", "", "The name of the application on the remote server")
	cmd.Flags().Bool("verbose", false, "Enable verbose output for SSH and rsync commands")
	cmd.Flags().Bool("quiet", false, "Suppress all output except for errors")
	cmd.Flags().Int("ssh-port", 22, "SSH port to use for the connection")
	cmd.Flags().StringP("ssh-key", "i", "", "Path to the SSH private key")

	_ = cmd.MarkFlagRequired("path")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("app")

	return cmd
}

// pushUI wraps pterm components to control output based on quiet flag.
type pushUI struct {
	spinner *pterm.SpinnerPrinter
	quiet   bool
}

func newPushUI(quiet bool) *pushUI {
	ui := &pushUI{quiet: quiet}
	if !quiet {
		// Assign a non-nil spinner so methods can be called.
		// Start it fresh for each major step.
		ui.spinner = &pterm.SpinnerPrinter{}
	}
	return ui
}

func (ui *pushUI) Start(text string) {
	if !ui.quiet {
		ui.spinner, _ = pterm.DefaultSpinner.Start(text)
	}
}

func (ui *pushUI) UpdateText(text string) {
	if !ui.quiet {
		ui.spinner.UpdateText(text)
	}
}

func (ui *pushUI) Success(text string) {
	if !ui.quiet {
		ui.spinner.Success(text)
	}
}

func (ui *pushUI) Fail(text string) {
	// Always print failures
	pterm.Error.Println(text)
	if !ui.quiet {
		ui.spinner.Fail()
	}
}

// preflightCheck checks if required local commands are installed.
func preflightCheck(commands ...string) error {
	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf(i18n.T().PreflightCheckFailed, cmd, err)
		}
	}
	return nil
}

func runPush(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	verbose, _ := cmd.Flags().GetBool("verbose")
	sshPort, _ := cmd.Flags().GetInt("ssh-port")
	sshKey, _ := cmd.Flags().GetString("ssh-key")
	sourceDir, _ := cmd.Flags().GetString("path")
	destination, _ := cmd.Flags().GetString("to")
	appName, _ := cmd.Flags().GetString("app")

	ui := newPushUI(quiet)
	ui.Start("Initializing deployment...")

	// Local environment pre-flight check
	ui.UpdateText("Performing local pre-flight checks...")
	if err := preflightCheck("ssh", "rsync"); err != nil {
		ui.Fail(err.Error())
		return err
	}
	ui.Success("Local checks passed.")

	user, host, err := parseDestination(destination)
	if err != nil {
		ui.Fail(err.Error())
		return err
	}

	opts := &core.PushOptions{
		SourceDir:    sourceDir,
		User:         user,
		Host:         host,
		Port:         sshPort,
		KeyFile:      sshKey,
		AppName:      appName,
		GetVersion:   GetVersion,
		NewSSHClient: newSSHClient,
		Quiet:        quiet,
		Verbose:      verbose,
	}

	pusher := core.NewPusher(opts)

	// This is a simplified representation. A real implementation would use channels
	// from the core logic to update the spinner with more granular steps.
	ui.UpdateText("Executing remote deployment workflow...")

	if err := pusher.Push(); err != nil {
		if err == core.ErrRemoteUpdated {
			// The core logic printed the "please run again" message.
			// We just mark the current operation as successful in its own context.
			ui.Success("Remote update complete. Please run the command again.")
			return nil // Exit gracefully
		}
		ui.Fail(err.Error())
		return err
	}

	ui.Success("Deployment workflow completed successfully!")

	return nil
}

// parseDestination splits a destination string like "user@host" into user and host.
// If user is not provided, it defaults to "" and ssh will use the config file.
func parseDestination(dest string) (user, host string, err error) {
	if dest == "" {
		return "", "", fmt.Errorf("destination cannot be empty")
	}

	if !strings.Contains(dest, "@") {
		// If no user is specified, we rely on the user's SSH config.
		return "", dest, nil
	}

	parts := strings.SplitN(dest, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid destination format: '%s'. Expected 'user@host' or 'host'", dest)
	}
	return parts[0], parts[1], nil
}
