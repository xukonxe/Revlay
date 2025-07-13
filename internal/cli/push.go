package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ssh"
)

var (
	// newSSHClient is a factory function for creating an ssh.Client.
	// It's a variable so it can be replaced in tests.
	newSSHClient = func(user, host string) ssh.Client {
		return ssh.NewClient(user, host)
	}
)

// NewPushCommand creates the `revlay push` command.
func NewPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source_dir> to <[user@]host>",
		Short: i18n.T().PushShortDesc,
		Long:  i18n.T().PushLongDesc,
		Example: `  # Push the './dist' directory to the 'production' app on the remote server
  revlay push ./dist to deploy@192.168.1.100 --to production`,
		// We need exactly 3 arguments: source, "to", destination
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return fmt.Errorf("requires exactly 3 arguments: <source_dir> to <[user@]host>")
			}
			if args[1] != "to" {
				return fmt.Errorf("invalid syntax. use 'revlay push <source_dir> to <[user@]host>'")
			}
			return nil
		},
		RunE: runPush,
	}

	cmd.Flags().String("to", "", "The name of the application on the remote server to deploy to")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runPush(cmd *cobra.Command, args []string) error {
	sourceDir := args[0]
	destination := args[2]
	appName, _ := cmd.Flags().GetString("to")

	user, host, err := parseDestination(destination)
	if err != nil {
		return err
	}

	fmt.Println(color.Cyan(i18n.T().PushStarting, destination, appName))

	client := newSSHClient(user, host)

	// Step 1: Pre-flight check - verify revlay is on remote
	fmt.Println(color.Cyan(i18n.T().PushCheckingRemote))
	if _, err := client.RunCommand("command -v revlay"); err != nil {
		return fmt.Errorf("revlay not found on the remote server. Please ensure it's installed and in the user's PATH")
	}
	fmt.Println(color.Green(i18n.T().PushRemoteFound))

	// Step 2: Create a temporary directory on the remote server
	fmt.Println(color.Cyan(i18n.T().PushCreatingTempDir))
	remoteTempDir, err := client.RunCommand("mktemp -d")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory on remote: %w", err)
	}
	remoteTempDir = strings.TrimSpace(remoteTempDir)
	fmt.Println(color.Green(i18n.T().PushTempDirCreated, remoteTempDir))

	// Defer cleanup of the temporary directory
	defer func() {
		fmt.Println(color.Cyan(i18n.T().PushCleaningUp))
		if _, err := client.RunCommand(fmt.Sprintf("rm -rf %s", remoteTempDir)); err != nil {
			fmt.Println(color.Red(i18n.T().PushCleanupFailed, remoteTempDir, err))
		} else {
			fmt.Println(color.Green(i18n.T().PushCleanupComplete))
		}
	}()

	// Step 3: Use rsync to push files
	fmt.Println(color.Cyan(i18n.T().PushSyncingFiles, remoteTempDir))
	if err := client.Rsync(sourceDir, remoteTempDir); err != nil {
		return fmt.Errorf("failed to rsync files: %w", err)
	}
	fmt.Println(color.Green(i18n.T().PushSyncComplete))

	// Step 4: Execute remote deploy
	fmt.Println(color.Cyan(i18n.T().PushTriggeringDeploy, appName))
	deployCommand := fmt.Sprintf("revlay deploy --from-dir %s %s", remoteTempDir, appName)
	if err := client.RunCommandStream(deployCommand); err != nil {
		return fmt.Errorf("remote deployment failed: %w", err)
	}

	fmt.Println(color.Green(i18n.T().PushComplete))

	return nil
}

// parseDestination splits a destination string like "user@host" into user and host.
// If user is not provided, it defaults to the current user.
func parseDestination(dest string) (user, host string, err error) {
	if dest == "" {
		return "", "", fmt.Errorf("destination cannot be empty")
	}

	if !strings.Contains(dest, "@") {
		// If no user is specified, we could default to the current OS user,
		// but for now, let's keep it simple and just use the host.
		// The user will rely on their SSH config.
		return "", dest, nil
	}

	parts := strings.SplitN(dest, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid destination format: '%s'. Expected 'user@host' or 'host'", dest)
	}
	return parts[0], parts[1], nil
}
