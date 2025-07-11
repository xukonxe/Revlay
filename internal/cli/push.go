package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewPushCommand creates the `revlay push` command.
func NewPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source_dir> to <[user@]host>",
		Short: "Push local directory to remote and deploy",
		Long: `This command uses rsync to push a local directory to a remote server 
and then triggers 'revlay deploy' on the remote machine.

It streamlines the deployment process by packaging, transferring, and activating
a new release in a single step.`,
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

	fmt.Printf("Parsed destination: user='%s', host='%s'\n", user, host)
	fmt.Printf("Source directory: %s\n", sourceDir)
	fmt.Printf("Target application: %s\n", appName)


	fmt.Println("\nPhase 2: Push command logic coming soon!")
	fmt.Println("Next steps:")
	fmt.Println("1. [Done] Parse destination string.")
	fmt.Println("2. Use SSH to check if 'revlay' is installed on the remote server.")
	fmt.Println("3. Use rsync to copy the source directory to a temporary location on the server.")
	fmt.Println("4. Execute 'revlay deploy --from-dir' on the server to start the deployment.")
	fmt.Println("5. Clean up the temporary directory on the server.")

	return nil
}

// parseDestination splits a destination string like "user@host" into user and host.
// If user is not provided, it defaults to the current user.
func parseDestination(dest string) (user, host string, err error) {
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