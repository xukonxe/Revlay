package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
)

// Execute is the main entry point for the CLI.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.Red("Error: %v", err))
		os.Exit(1)
	}
}

// newRootCmd creates the root command and adds all subcommands.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revlay",
		Short: "Revlay is a simple and powerful deployment tool.",
		Long: `A fast, reliable, and easy-to-use deployment tool that helps you
automate the process of releasing your applications.`,
		// Silence errors, we'll handle them in Execute()
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Add all the commands
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewDeployCommand())
	cmd.AddCommand(NewRollbackCommand())
	cmd.AddCommand(NewReleasesCommand())
	cmd.AddCommand(NewStatusCommand())
	cmd.AddCommand(NewPushCommand())

	// Add a persistent flag for the config file to the root command.
	// This makes it available to all subcommands.
	cmd.PersistentFlags().StringP("config", "c", "", "Path to config file (default is revlay.yml)")

	return cmd
} 